package main

import (
	"context"
	"strings"
	"testing"
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
