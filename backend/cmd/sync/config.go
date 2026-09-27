package main

import (
	"errors"
	"os"
)

type config struct {
	databaseURL string
	twitch      twitchConfig
}

type twitchConfig struct {
	clientID     string
	clientSecret string
}

func (c twitchConfig) configured() bool {
	return c.clientID != "" && c.clientSecret != ""
}

func loadConfig() (config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}

	return config{
		databaseURL: databaseURL,
		twitch: twitchConfig{
			clientID:     os.Getenv("TWITCH_CLIENT_ID"),
			clientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		},
	}, nil
}
