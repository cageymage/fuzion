package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type config struct {
	addr           string
	databaseURL    string
	allowedOrigins []string
	discord        discordConfig
	youtube        youTubeConfig

	// Optional: empty disables the Discord ping for new applications, so local
	// dev and CI need no real channel.
	recruitingWebhookURL string

	// Optional: empty disables the Discord cross-post when a news post is published.
	announcementsWebhookURL string

	// Public address of the site, used for links in announcements, e.g. http://localhost:5173 in dev.
	siteBaseURL string
}

// Both values are optional: without them the Home page simply has no suggested
// video to show, which local dev and forks should not have to set up.
type youTubeConfig struct {
	apiKey     string
	playlistID string
}

func (c youTubeConfig) enabled() bool {
	return c.apiKey != "" && c.playlistID != ""
}

type discordConfig struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

func loadConfig() (config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
		// Render injects PORT rather than letting the service pick its own.
		if port := os.Getenv("PORT"); port != "" {
			addr = ":" + port
		}
	}

	allowedOrigins := []string{"http://localhost:5173"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}

	discord, err := loadDiscordConfig()
	if err != nil {
		return config{}, err
	}

	if os.Getenv("DISCORD_ANNOUNCEMENTS_WEBHOOK_URL") != "" && os.Getenv("SITE_BASE_URL") == "" {
		return config{}, errors.New("SITE_BASE_URL is required when DISCORD_ANNOUNCEMENTS_WEBHOOK_URL is set")
	}

	return config{
		addr:           addr,
		databaseURL:    databaseURL,
		allowedOrigins: allowedOrigins,
		discord:        discord,
		youtube: youTubeConfig{
			apiKey:     os.Getenv("YOUTUBE_API_KEY"),
			playlistID: os.Getenv("YOUTUBE_PLAYLIST_ID"),
		},
		recruitingWebhookURL:    os.Getenv("DISCORD_RECRUITING_WEBHOOK_URL"),
		announcementsWebhookURL: os.Getenv("DISCORD_ANNOUNCEMENTS_WEBHOOK_URL"),
		siteBaseURL:             os.Getenv("SITE_BASE_URL"),
	}, nil
}

// A mis-deployed API should refuse to start rather than 500 on every login attempt.
func loadDiscordConfig() (discordConfig, error) {
	cfg := discordConfig{
		clientID:     os.Getenv("DISCORD_CLIENT_ID"),
		clientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		redirectURL:  os.Getenv("DISCORD_REDIRECT_URL"),
	}
	for _, v := range []struct{ name, value string }{
		{"DISCORD_CLIENT_ID", cfg.clientID},
		{"DISCORD_CLIENT_SECRET", cfg.clientSecret},
		{"DISCORD_REDIRECT_URL", cfg.redirectURL},
	} {
		if v.value == "" {
			return discordConfig{}, fmt.Errorf("%s is required", v.name)
		}
	}
	return cfg, nil
}
