package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// The unreachable database URL proves these paths return before connecting.
const unreachableDatabase = "postgres://nobody@127.0.0.1:1/none"

func TestRunBlizzardRoster_DoesNothing_WhenFlagIsDisabled(t *testing.T) {
	// given the flag is off and Blizzard credentials are missing
	cfg := config{databaseURL: unreachableDatabase}

	// when the roster sync runs
	err := runBlizzardRoster(context.Background(), cfg)

	// then I expect it to exit cleanly without touching the database
	if err != nil {
		t.Errorf("runBlizzardRoster = %v, want nil", err)
	}
}

func TestRunBlizzardRoster_ReturnsError_WhenFlagIsEnabledWithoutCredentials(t *testing.T) {
	// given the flag is on but no credentials or guild are configured
	cfg := config{databaseURL: unreachableDatabase, blizzard: blizzardConfig{enabled: true}}

	// when the roster sync runs
	err := runBlizzardRoster(context.Background(), cfg)

	// then I expect an error naming what is missing
	if err == nil {
		t.Fatal("runBlizzardRoster = nil, want an error")
	}
	for _, name := range []string{"BLIZZARD_CLIENT_ID", "BLIZZARD_CLIENT_SECRET", "GUILD_REALM_SLUG", "GUILD_NAME_SLUG"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention %s", err, name)
		}
	}
}

func TestRunYouTube_DoesNothing_WhenAPIKeyIsMissing(t *testing.T) {
	// given no YouTube API key is configured
	cfg := config{databaseURL: unreachableDatabase}

	// when the YouTube sync runs
	err := runYouTube(context.Background(), cfg)

	// then I expect it to exit cleanly without touching the database
	if err != nil {
		t.Errorf("runYouTube = %v, want nil", err)
	}
}

func TestLoadConfig_ReadsAltRanks_WhenBlizzardAltRanksIsSet(t *testing.T) {
	// given alt ranks written with spaces around the numbers
	t.Setenv("DATABASE_URL", unreachableDatabase)
	t.Setenv("BLIZZARD_ALT_RANKS", " 4, 5 ,7")

	// when the config loads
	cfg, err := loadConfig()

	// then I expect the rank numbers in order
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if diff := cmp.Diff([]int{4, 5, 7}, cfg.blizzard.altRanks); diff != "" {
		t.Errorf("unexpected alt ranks (-want +got):\n%s", diff)
	}
}

func TestLoadConfig_ReadsNoAltRanks_WhenBlizzardAltRanksIsUnset(t *testing.T) {
	// given no alt ranks
	t.Setenv("DATABASE_URL", unreachableDatabase)
	t.Setenv("BLIZZARD_ALT_RANKS", "")

	// when the config loads
	cfg, err := loadConfig()

	// then I expect no alt ranks, so the sync leaves main/alt alone
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.blizzard.altRanks != nil {
		t.Errorf("altRanks = %v, want nil", cfg.blizzard.altRanks)
	}
}

func TestLoadConfig_ReturnsError_WhenBlizzardAltRanksHasANonNumber(t *testing.T) {
	// given an alt rank written as a name instead of a number
	t.Setenv("DATABASE_URL", unreachableDatabase)
	t.Setenv("BLIZZARD_ALT_RANKS", "4,Alt")

	// when the config loads
	_, err := loadConfig()

	// then I expect an error naming the variable and the bad value
	if err == nil {
		t.Fatal("loadConfig = nil, want an error")
	}
	if want := `BLIZZARD_ALT_RANKS: "Alt" is not a guild rank number`; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}
