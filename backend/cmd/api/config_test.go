package main

import "testing"

func setRequiredConfig(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("DISCORD_CLIENT_ID", "id")
	t.Setenv("DISCORD_CLIENT_SECRET", "secret")
	t.Setenv("DISCORD_REDIRECT_URL", "http://localhost/callback")
	t.Setenv("DISCORD_GUILD_ID", "")
	t.Setenv("DISCORD_ANNOUNCEMENTS_WEBHOOK_URL", "")
	t.Setenv("DISCORD_RECRUITING_WEBHOOK_URL", "")
	t.Setenv("SITE_BASE_URL", "")
	t.Setenv("BATTLENET_REDIRECT_URL", "")
	t.Setenv("BLIZZARD_CLIENT_ID", "")
	t.Setenv("BLIZZARD_CLIENT_SECRET", "")
	t.Setenv("BLIZZARD_REGION", "")
	t.Setenv("BLIZZARD_NAMESPACE", "")
	t.Setenv("SUBMISSIONS_ENABLED", "")
	t.Setenv("SUBMISSIONS_STORAGE_LIMIT_MB", "")
}

func TestLoadConfig_ReturnsError_WhenBattleNetRedirectIsSetWithoutBlizzardCredentials(t *testing.T) {
	// given a Battle.net redirect URL and no Blizzard client credentials
	setRequiredConfig(t)
	t.Setenv("BATTLENET_REDIRECT_URL", "http://localhost/battlenet/callback")

	// when I load the config
	_, err := loadConfig()

	// then I expect an error naming the missing variable
	want := "BLIZZARD_CLIENT_ID is required when BATTLENET_REDIRECT_URL is set"
	if err == nil || err.Error() != want {
		t.Errorf("expected error %q, got %v", want, err)
	}
}

func TestLoadConfig_LeavesBattleNetDisabled_WhenRedirectIsNotSet(t *testing.T) {
	// given no Battle.net redirect URL
	setRequiredConfig(t)

	// when I load the config
	cfg, err := loadConfig()

	// then I expect it to load with Battle.net linking disabled
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.battleNet.enabled() {
		t.Error("expected Battle.net linking to be disabled")
	}
}

func TestLoadConfig_ReturnsError_WhenRecruitingWebhookIsSetWithoutSiteBaseURL(t *testing.T) {
	// given a recruiting webhook and no site base URL
	setRequiredConfig(t)
	t.Setenv("DISCORD_RECRUITING_WEBHOOK_URL", "https://discord.example/webhook")

	// when I load the config
	_, err := loadConfig()

	// then I expect an error naming the missing variable
	want := "SITE_BASE_URL is required when DISCORD_RECRUITING_WEBHOOK_URL is set"
	if err == nil || err.Error() != want {
		t.Errorf("expected error %q, got %v", want, err)
	}
}

func TestLoadConfig_Succeeds_WhenRecruitingWebhookAndSiteBaseURLAreSet(t *testing.T) {
	// given a recruiting webhook and a site base URL
	setRequiredConfig(t)
	t.Setenv("DISCORD_RECRUITING_WEBHOOK_URL", "https://discord.example/webhook")
	t.Setenv("SITE_BASE_URL", "https://fuzion.example")

	// when I load the config
	cfg, err := loadConfig()

	// then I expect it to load with the site base URL
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.siteBaseURL != "https://fuzion.example" {
		t.Errorf("expected siteBaseURL %q, got %q", "https://fuzion.example", cfg.siteBaseURL)
	}
}

func TestLoadConfig_ReadsDiscordGuildID_WhenItIsSet(t *testing.T) {
	// given the guild's Discord server ID in the environment
	setRequiredConfig(t)
	t.Setenv("DISCORD_GUILD_ID", "987654321098765432")

	// when I load the config
	cfg, err := loadConfig()

	// then I expect the guild ID to be passed along for the membership check
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.discord.guildID != "987654321098765432" {
		t.Errorf("expected guild ID 987654321098765432, got %q", cfg.discord.guildID)
	}
}

func TestLoadConfig_DefaultsBattleNetNamespaceToTheRegionsProfile_WhenNamespaceIsNotSet(t *testing.T) {
	// given Battle.net linking in the eu region with no namespace
	setRequiredConfig(t)
	t.Setenv("BATTLENET_REDIRECT_URL", "http://localhost/battlenet/callback")
	t.Setenv("BLIZZARD_CLIENT_ID", "id")
	t.Setenv("BLIZZARD_CLIENT_SECRET", "secret")
	t.Setenv("BLIZZARD_REGION", "eu")

	// when I load the config
	cfg, err := loadConfig()

	// then I expect the eu region with the eu profile namespace, as the roster sync defaults it
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.battleNet.region != "eu" || cfg.battleNet.namespace != "profile-eu" {
		t.Errorf("region, namespace = %q, %q, want %q, %q", cfg.battleNet.region, cfg.battleNet.namespace, "eu", "profile-eu")
	}
}

func TestLoadConfig_OpensSubmissionsWith512MiB_WhenTheirVariablesAreUnset(t *testing.T) {
	// given neither submissions variable
	setRequiredConfig(t)

	// when I load the config
	cfg, err := loadConfig()

	// then I expect submissions on, so a missing variable never closes them, with the default ceiling
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.submissions != (submissionsConfig{enabled: true, storageLimitBytes: 512 << 20}) {
		t.Errorf("submissions = %+v", cfg.submissions)
	}
}

func TestLoadConfig_ClosesSubmissions_WhenSubmissionsEnabledIsFalse(t *testing.T) {
	// given the emergency switch turned off and a 64 MiB ceiling
	setRequiredConfig(t)
	t.Setenv("SUBMISSIONS_ENABLED", "false")
	t.Setenv("SUBMISSIONS_STORAGE_LIMIT_MB", "64")

	// when I load the config
	cfg, err := loadConfig()

	// then I expect submissions off, with the ceiling I set
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.submissions != (submissionsConfig{enabled: false, storageLimitBytes: 64 << 20}) {
		t.Errorf("submissions = %+v", cfg.submissions)
	}
}

func TestLoadConfig_ReturnsError_WhenSubmissionsEnabledIsNotABoolean(t *testing.T) {
	// given a mistyped switch
	setRequiredConfig(t)
	t.Setenv("SUBMISSIONS_ENABLED", "nope")

	// when I load the config
	_, err := loadConfig()

	// then I expect startup to fail naming the variable, rather than guess
	want := `SUBMISSIONS_ENABLED must be true or false, got "nope"`
	if err == nil || err.Error() != want {
		t.Errorf("expected error %q, got %v", want, err)
	}
}
