package auth_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func newDiscord(baseURL string) *auth.Discord {
	return auth.NewDiscord(auth.DiscordConfig{
		ClientID:     testutil.DiscordClientID,
		ClientSecret: testutil.DiscordClientSecret,
		RedirectURL:  testutil.DiscordRedirectURL,
		BaseURL:      baseURL,
	}, http.DefaultClient)
}

func TestDiscordAuthURL_IncludesClientIDRedirectAndState(t *testing.T) {
	// given a Discord provider configured against a known base URL
	discord := newDiscord("https://discord.com/api")

	// when I build the authorize URL for a state
	got := discord.AuthURL("random-state-123")

	// then I expect Discord's authorize endpoint with every required query param
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse auth url %q: %v", got, err)
	}
	if parsed.Scheme+"://"+parsed.Host+parsed.Path != "https://discord.com/api/oauth2/authorize" {
		t.Errorf("unexpected authorize endpoint: %s", got)
	}
	wantQuery := url.Values{
		"client_id":     {testutil.DiscordClientID},
		"redirect_uri":  {testutil.DiscordRedirectURL},
		"response_type": {"code"},
		"scope":         {"identify"},
		"state":         {"random-state-123"},
	}
	if diff := cmp.Diff(wantQuery, parsed.Query()); diff != "" {
		t.Errorf("unexpected query params (-want +got):\n%s", diff)
	}
}

func TestDiscordExchange_ReturnsIdentity_WhenDiscordAcceptsTheCode(t *testing.T) {
	// given a fake Discord that accepts the code and knows the user
	fake := testutil.NewFakeDiscord(t)
	avatar := "8342729096ea3675442027381ff50dfe"
	fake.GrantCode("abc123", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane", Avatar: &avatar})
	discord := newDiscord(fake.URL)

	// when I exchange the code
	got, err := discord.Exchange(context.Background(), "abc123")

	// then I expect the user's identity with a CDN avatar URL
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	want := auth.Identity{
		ProviderUserID: "80351110224678912",
		Username:       "thundermane",
		AvatarURL:      "https://cdn.discordapp.com/avatars/80351110224678912/8342729096ea3675442027381ff50dfe.png",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected identity (-want +got):\n%s", diff)
	}
}

func TestDiscordExchange_ReturnsEmptyAvatarURL_WhenUserHasNoAvatar(t *testing.T) {
	// given a fake Discord whose user has a null avatar
	fake := testutil.NewFakeDiscord(t)
	fake.GrantCode("abc123", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane", Avatar: nil})
	discord := newDiscord(fake.URL)

	// when I exchange the code
	got, err := discord.Exchange(context.Background(), "abc123")

	// then I expect an identity with no avatar URL
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	want := auth.Identity{
		ProviderUserID: "80351110224678912",
		Username:       "thundermane",
		AvatarURL:      "",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected identity (-want +got):\n%s", diff)
	}
}

func TestDiscordExchange_ReturnsError_WhenTokenEndpointRejectsTheCode(t *testing.T) {
	// given a fake Discord that was never told about this code
	fake := testutil.NewFakeDiscord(t)
	discord := newDiscord(fake.URL)

	// when I exchange the code
	got, err := discord.Exchange(context.Background(), "never-issued")

	// then I expect a wrapped error naming the token step and an empty identity
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "exchange discord code") || !strings.Contains(err.Error(), "400") {
		t.Errorf("unexpected error: %v", err)
	}
	if got != (auth.Identity{}) {
		t.Errorf("expected empty identity, got %+v", got)
	}
}

func TestDiscordExchange_ReturnsError_WhenUserEndpointFails(t *testing.T) {
	// given a fake Discord that issues a token but fails the user lookup
	fake := testutil.NewFakeDiscord(t)
	fake.GrantCode("abc123", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})
	fake.UserEndpointStatus = http.StatusInternalServerError
	discord := newDiscord(fake.URL)

	// when I exchange the code
	got, err := discord.Exchange(context.Background(), "abc123")

	// then I expect a wrapped error naming the user step and an empty identity
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "fetch discord user") || !strings.Contains(err.Error(), "500") {
		t.Errorf("unexpected error: %v", err)
	}
	if got != (auth.Identity{}) {
		t.Errorf("expected empty identity, got %+v", got)
	}
}
