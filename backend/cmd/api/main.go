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

	"github.com/cageymage/fuzion/backend/internal/applications"
	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/clock"
	idiscord "github.com/cageymage/fuzion/backend/internal/discord"
	"github.com/cageymage/fuzion/backend/internal/images"
	"github.com/cageymage/fuzion/backend/internal/news"
	"github.com/cageymage/fuzion/backend/internal/professions"
	"github.com/cageymage/fuzion/backend/internal/raidprogress"
	"github.com/cageymage/fuzion/backend/internal/raids"
	"github.com/cageymage/fuzion/backend/internal/roster"
	"github.com/cageymage/fuzion/backend/internal/server"
	"github.com/cageymage/fuzion/backend/internal/streams"
	"github.com/cageymage/fuzion/backend/internal/turnstile"
	"github.com/cageymage/fuzion/backend/internal/youtube"
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

	// Assign only when configured: a typed-nil *BattleNet in the option would read as enabled.
	var authOptions []auth.ServiceOption
	if cfg.battleNet.enabled() {
		authOptions = append(authOptions, auth.WithBattleNet(auth.NewBattleNet(auth.BattleNetConfig{
			ClientID:     cfg.battleNet.clientID,
			ClientSecret: cfg.battleNet.clientSecret,
			RedirectURL:  cfg.battleNet.redirectURL,
			BaseURL:      auth.BattleNetAPIBaseURL,
		}, &http.Client{Timeout: 10 * time.Second})))
	} else {
		slog.Info("battle.net linking disabled: BATTLENET_REDIRECT_URL is not set")
	}

	playlistID := cfg.youtube.playlistID
	if !cfg.youtube.enabled() {
		slog.Info("suggested videos disabled: YOUTUBE_API_KEY and YOUTUBE_PLAYLIST_ID are not both set")
		playlistID = ""
	}
	youTube := youtube.NewClient(youtube.Config{
		APIKey:  cfg.youtube.apiKey,
		BaseURL: youtube.APIBaseURL,
	}, &http.Client{Timeout: 10 * time.Second})

	// Assign only when set: a typed-nil *Webhook in the interface would not read as disabled.
	var recruiting applications.Notifier
	if cfg.recruitingWebhookURL != "" {
		recruiting = idiscord.NewWebhook(cfg.recruitingWebhookURL, &http.Client{Timeout: 5 * time.Second})
	} else {
		slog.Info("application notifications disabled: DISCORD_RECRUITING_WEBHOOK_URL is not set")
	}

	// Same typed-nil pitfall: assign only when the secret is set.
	var botCheck applications.Verifier
	if cfg.turnstileSecretKey != "" {
		botCheck = turnstile.NewClient(turnstile.Config{
			SecretKey: cfg.turnstileSecretKey,
			BaseURL:   turnstile.VerifyBaseURL,
		}, &http.Client{Timeout: 5 * time.Second})
	} else {
		slog.Info("application bot check disabled: TURNSTILE_SECRET_KEY is not set")
	}

	var announcements news.Notifier
	if cfg.announcementsWebhookURL != "" {
		announcements = idiscord.NewWebhook(cfg.announcementsWebhookURL, &http.Client{Timeout: 5 * time.Second})
	} else {
		slog.Info("news announcements disabled: DISCORD_ANNOUNCEMENTS_WEBHOOK_URL is not set")
	}

	router := server.New(server.Deps{
		Applications:   applications.NewHandler(applications.NewService(applications.NewRepo(db), clock.System{}, recruiting, botCheck, cfg.siteBaseURL)),
		Auth:           auth.NewHandler(auth.NewService(discord, auth.NewRepo(db), authOptions...)),
		Images:         images.NewHandler(images.NewService(images.NewRepo(db))),
		News:           news.NewHandler(news.NewService(news.NewRepo(db), clock.System{}, announcements, cfg.siteBaseURL), news.NewShellFetcher(cfg.siteBaseURL+"/index.html", &http.Client{Timeout: 5 * time.Second}, clock.System{})),
		Professions:    professions.NewHandler(professions.NewService(professions.NewRepo(db))),
		RaidProgress:   raidprogress.NewHandler(raidprogress.NewService(raidprogress.NewRepo(db), clock.System{})),
		Raids:          raids.NewHandler(raids.NewService(raids.NewRepo(db))),
		Roster:         roster.NewHandler(roster.NewService(roster.NewRepo(db))),
		Streams:        streams.NewHandler(streams.NewService(streams.NewRepo(db)), streams.NewSuggestedVideos(youTube, playlistID, clock.System{})),
		AllowedOrigins: cfg.allowedOrigins,
	})

	return server.Run(ctx, cfg.addr, router)
}
