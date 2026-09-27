package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cageymage/fuzion/backend/internal/streams"
	"github.com/cageymage/fuzion/backend/internal/synclog"
	"github.com/cageymage/fuzion/backend/internal/twitch"
)

const usage = "usage: sync twitch"

func main() {
	if len(os.Args) != 2 || os.Args[1] != "twitch" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	if err := runTwitch(); err != nil {
		slog.Error("sync twitch failed", "error", err)
		os.Exit(1)
	}
}

func runTwitch() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if !cfg.twitch.configured() {
		slog.Info("sync twitch skipped: TWITCH_CLIENT_ID and TWITCH_CLIENT_SECRET are not both set")
		return nil
	}

	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	client := twitch.NewClient(twitch.Config{
		ClientID:     cfg.twitch.clientID,
		ClientSecret: cfg.twitch.clientSecret,
		BaseURL:      twitch.APIBaseURL,
		AuthURL:      twitch.AuthBaseURL,
	}, &http.Client{Timeout: 10 * time.Second})

	return streams.NewSyncTwitch(streams.NewRepo(db), client, synclog.NewRepo(db)).Run(ctx)
}
