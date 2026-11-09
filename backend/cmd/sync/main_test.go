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
	for _, name := range []string{"BLIZZARD_CLIENT_ID", "BLIZZARD_CLIENT_SECRET", "GUILD_NAME_SLUG"} {
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

func TestLoadConfig_DefaultsBlizzardNamespaceToTheRegionsForeverProfile_WhenNamespaceIsNotSet(t *testing.T) {
	// given the eu region with no namespace
	t.Setenv("DATABASE_URL", unreachableDatabase)
	t.Setenv("BLIZZARD_REGION", "eu")
	t.Setenv("BLIZZARD_NAMESPACE", "")

	// when the config shared by every sync job loads
	cfg, err := loadConfig()

	// then I expect Forever's eu profile namespace
	if err != nil {
		t.Fatalf("loadConfig = %v, want nil", err)
	}
	if cfg.blizzard.namespace != "profile-forever-eu" {
		t.Errorf("namespace = %q, want %q", cfg.blizzard.namespace, "profile-forever-eu")
	}
}

func TestSyncProfessions_IsNoOp_WhenFlagIsOff(t *testing.T) {
	// given the professions flag is off and Blizzard credentials are missing
	cfg := config{databaseURL: unreachableDatabase}

	// when the professions sync runs
	err := runBlizzardProfessions(context.Background(), cfg)

	// then I expect it to exit cleanly without touching the database
	if err != nil {
		t.Errorf("runBlizzardProfessions = %v, want nil", err)
	}
}

func TestSyncProfessions_ReturnsError_WhenFlagIsOnWithoutCredentials(t *testing.T) {
	// given the professions flag is on but no credentials are configured
	cfg := config{databaseURL: unreachableDatabase, blizzard: blizzardConfig{professionsEnabled: true}}

	// when the professions sync runs
	err := runBlizzardProfessions(context.Background(), cfg)

	// then I expect an error naming the missing credentials
	if err == nil {
		t.Fatal("runBlizzardProfessions = nil, want an error")
	}
	for _, name := range []string{"BLIZZARD_CLIENT_ID", "BLIZZARD_CLIENT_SECRET"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention %s", err, name)
		}
	}
}
