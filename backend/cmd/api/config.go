package main

import (
	"errors"
	"os"
	"strings"
)

type config struct {
	addr           string
	databaseURL    string
	allowedOrigins []string
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

	return config{addr: addr, databaseURL: databaseURL, allowedOrigins: allowedOrigins}, nil
}
