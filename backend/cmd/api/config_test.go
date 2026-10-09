package main

import "testing"

func setRequiredConfig(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("DISCORD_CLIENT_ID", "id")
	t.Setenv("DISCORD_CLIENT_SECRET", "secret")
	t.Setenv("DISCORD_REDIRECT_URL", "http://localhost/callback")
	t.Setenv("DISCORD_ANNOUNCEMENTS_WEBHOOK_URL", "")
	t.Setenv("DISCORD_RECRUITING_WEBHOOK_URL", "")
	t.Setenv("SITE_BASE_URL", "")
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
