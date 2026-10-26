package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type config struct {
	databaseURL string
	twitch      twitchConfig
	youTube     youTubeConfig
	blizzard    blizzardConfig
}

type twitchConfig struct {
	clientID     string
	clientSecret string
}

func (c twitchConfig) configured() bool {
	return c.clientID != "" && c.clientSecret != ""
}

type youTubeConfig struct {
	apiKey string
}

func (c youTubeConfig) configured() bool {
	return c.apiKey != ""
}

type blizzardConfig struct {
	enabled      bool
	clientID     string
	clientSecret string
	region       string
	namespace    string
	realmSlug    string
	guildSlug    string
	altRanks     []int
}

// The job is flagged off until Blizzard's API is known to serve Forever. Once it is
// switched on, a missing setting is a mistake worth failing loudly for.
func (c blizzardConfig) validate() error {
	required := []struct{ name, value string }{
		{"BLIZZARD_CLIENT_ID", c.clientID},
		{"BLIZZARD_CLIENT_SECRET", c.clientSecret},
		{"GUILD_REALM_SLUG", c.realmSlug},
		{"GUILD_NAME_SLUG", c.guildSlug},
	}
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("SYNC_BLIZZARD_ROSTER_ENABLED is true but these are not set: %s", strings.Join(missing, ", "))
	}
	return nil
}

func loadConfig() (config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}

	altRanks, err := parseRanks(os.Getenv("BLIZZARD_ALT_RANKS"))
	if err != nil {
		return config{}, fmt.Errorf("BLIZZARD_ALT_RANKS: %w", err)
	}

	region := envOr("BLIZZARD_REGION", "us")
	return config{
		databaseURL: databaseURL,
		twitch: twitchConfig{
			clientID:     os.Getenv("TWITCH_CLIENT_ID"),
			clientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		},
		youTube: youTubeConfig{
			apiKey: os.Getenv("YOUTUBE_API_KEY"),
		},
		blizzard: blizzardConfig{
			enabled:      os.Getenv("SYNC_BLIZZARD_ROSTER_ENABLED") == "true",
			clientID:     os.Getenv("BLIZZARD_CLIENT_ID"),
			clientSecret: os.Getenv("BLIZZARD_CLIENT_SECRET"),
			region:       region,
			namespace:    envOr("BLIZZARD_NAMESPACE", "profile-"+region),
			realmSlug:    os.Getenv("GUILD_REALM_SLUG"),
			guildSlug:    os.Getenv("GUILD_NAME_SLUG"),
			altRanks:     altRanks,
		},
	}, nil
}

// "4, 5" becomes [4 5]; an empty value means no alt ranks.
func parseRanks(value string) ([]int, error) {
	var ranks []int
	for _, field := range strings.Split(value, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		rank, err := strconv.Atoi(field)
		if err != nil || rank < 0 {
			return nil, fmt.Errorf("%q is not a guild rank number", field)
		}
		ranks = append(ranks, rank)
	}
	return ranks, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
