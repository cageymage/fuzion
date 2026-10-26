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

func newDiscordInGuild(baseURL string) *auth.Discord {
	return auth.NewDiscord(auth.DiscordConfig{
		ClientID:     testutil.DiscordClientID,
		ClientSecret: testutil.DiscordClientSecret,
		RedirectURL:  testutil.DiscordRedirectURL,
		BaseURL:      baseURL,
		GuildID:      testutil.DiscordGuildID,
	}, http.DefaultClient)
}

func TestDiscordAuthURL_RequestsGuildsScope_WhenGuildIDIsConfigured(t *testing.T) {
	// given a Discord provider that knows which server to look for
	discord := newDiscordInGuild("https://discord.com/api")

	// when I build the authorize URL
	got := discord.AuthURL("random-state-123")

	// then I expect the scope to ask for the user's server list as well as their identity
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse auth url %q: %v", got, err)
	}
	if scope := parsed.Query().Get("scope"); scope != "identify guilds" {
		t.Errorf("expected scope %q, got %q", "identify guilds", scope)
	}
}

func TestDiscordExchange_ReportsServerMembership_WhenUserIsInTheConfiguredGuild(t *testing.T) {
	// given a user who is in the guild's Discord server and one other
	fake := testutil.NewFakeDiscord(t)
	fake.GrantCode("abc123", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane", Guilds: []string{"111111111111111111", testutil.DiscordGuildID}})
	discord := newDiscordInGuild(fake.URL)

	// when I exchange the code
	got, err := discord.Exchange(context.Background(), "abc123")

	// then I expect the identity to say they are in the server
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	want := auth.Identity{ProviderUserID: "80351110224678912", Username: "thundermane", InDiscordServer: true}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected identity (-want +got):\n%s", diff)
	}
}

func TestDiscordExchange_ReportsNoServerMembership_WhenUserIsNotInTheConfiguredGuild(t *testing.T) {
	// given a user who is only in other servers
	fake := testutil.NewFakeDiscord(t)
	fake.GrantCode("abc123", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane", Guilds: []string{"111111111111111111"}})
	discord := newDiscordInGuild(fake.URL)

	// when I exchange the code
	got, err := discord.Exchange(context.Background(), "abc123")

	// then I expect the identity to say they are not in the server
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	want := auth.Identity{ProviderUserID: "80351110224678912", Username: "thundermane", InDiscordServer: false}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected identity (-want +got):\n%s", diff)
	}
}

func TestDiscordExchange_ReturnsError_WhenGuildsEndpointFails(t *testing.T) {
	// given a fake Discord that fails the server list lookup
	fake := testutil.NewFakeDiscord(t)
	fake.GrantCode("abc123", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})
	fake.GuildsEndpointStatus = http.StatusInternalServerError
	discord := newDiscordInGuild(fake.URL)

	// when I exchange the code
	got, err := discord.Exchange(context.Background(), "abc123")

	// then I expect a wrapped error naming the guilds step and an empty identity
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "fetch discord guilds") || !strings.Contains(err.Error(), "500") {
		t.Errorf("unexpected error: %v", err)
	}
	if got != (auth.Identity{}) {
		t.Errorf("expected empty identity, got %+v", got)
	}
}
