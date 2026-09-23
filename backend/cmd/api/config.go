package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type config struct {
	addr                    string
	databaseURL             string
	allowedOrigins          []string
	discord                 discordConfig
	bootstrapAdminDiscordID string
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

	return config{
		addr:                    addr,
		databaseURL:             databaseURL,
		allowedOrigins:          allowedOrigins,
		discord:                 discord,
		bootstrapAdminDiscordID: os.Getenv("BOOTSTRAP_ADMIN_DISCORD_ID"),
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
