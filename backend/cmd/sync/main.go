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

	"github.com/cageymage/fuzion/backend/internal/blizzard"
	"github.com/cageymage/fuzion/backend/internal/roster"
	"github.com/cageymage/fuzion/backend/internal/streams"
	"github.com/cageymage/fuzion/backend/internal/synclog"
	"github.com/cageymage/fuzion/backend/internal/twitch"
	"github.com/cageymage/fuzion/backend/internal/youtube"
)

const usage = "usage: sync twitch|youtube|blizzard-roster"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	var run func(context.Context, config) error
	switch os.Args[1] {
	case "twitch":
		run = runTwitch
	case "youtube":
		run = runYouTube
	case "blizzard-roster":
		run = runBlizzardRoster
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("sync failed", "job", os.Args[1], "error", fmt.Errorf("load config: %w", err))
		os.Exit(1)
	}
	if err := run(ctx, cfg); err != nil {
		slog.Error("sync failed", "job", os.Args[1], "error", err)
		os.Exit(1)
	}
}

func runTwitch(ctx context.Context, cfg config) error {
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

func runYouTube(ctx context.Context, cfg config) error {
	if !cfg.youTube.configured() {
		slog.Info("sync youtube skipped: YOUTUBE_API_KEY is not set")
		return nil
	}

	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	client := youtube.NewClient(youtube.Config{
		APIKey:  cfg.youTube.apiKey,
		BaseURL: youtube.APIBaseURL,
	}, &http.Client{Timeout: 10 * time.Second})

	return streams.NewSyncYouTube(streams.NewRepo(db), client, synclog.NewRepo(db)).Run(ctx)
}

func runBlizzardRoster(ctx context.Context, cfg config) error {
	if !cfg.blizzard.enabled {
		slog.Info("sync blizzard-roster disabled: SYNC_BLIZZARD_ROSTER_ENABLED is not true")
		return nil
	}
	if err := cfg.blizzard.validate(); err != nil {
		return err
	}
	altRanks, err := cfg.blizzard.parsedAltRanks()
	if err != nil {
		return err
	}

	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	client := blizzard.NewClient(blizzard.Config{
		ClientID:     cfg.blizzard.clientID,
		ClientSecret: cfg.blizzard.clientSecret,
		BaseURL:      blizzard.APIBaseURL(cfg.blizzard.region),
		AuthURL:      blizzard.AuthBaseURL,
		Namespace:    cfg.blizzard.namespace,
	}, &http.Client{Timeout: 30 * time.Second})

	return roster.NewSyncBlizzardRoster(roster.NewRepo(db), client, synclog.NewRepo(db), cfg.blizzard.realmSlug, cfg.blizzard.guildSlug, altRanks).Run(ctx)
}
