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

	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/news"
	"github.com/cageymage/fuzion/backend/internal/raids"
	"github.com/cageymage/fuzion/backend/internal/roster"
	"github.com/cageymage/fuzion/backend/internal/server"
	"github.com/cageymage/fuzion/backend/internal/streams"
	"github.com/cageymage/fuzion/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := migrations.Apply(cfg.databaseURL); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	discord := auth.NewDiscord(auth.DiscordConfig{
		ClientID:     cfg.discord.clientID,
		ClientSecret: cfg.discord.clientSecret,
		RedirectURL:  cfg.discord.redirectURL,
		BaseURL:      auth.DiscordAPIBaseURL,
	}, &http.Client{Timeout: 10 * time.Second})

	router := server.New(server.Deps{
		Auth:           auth.NewHandler(auth.NewService(discord, auth.NewRepo(db))),
		News:           news.NewHandler(news.NewService(news.NewRepo(db))),
		Raids:          raids.NewHandler(raids.NewService(raids.NewRepo(db))),
		Roster:         roster.NewHandler(roster.NewService(roster.NewRepo(db))),
		Streams:        streams.NewHandler(streams.NewService(streams.NewRepo(db))),
		AllowedOrigins: cfg.allowedOrigins,
	})

	return server.Run(ctx, cfg.addr, router)
}
