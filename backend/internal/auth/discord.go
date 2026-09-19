package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	DiscordAPIBaseURL = "https://discord.com/api"
	discordCDN        = "https://cdn.discordapp.com"
)

type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	BaseURL      string
}

var _ Provider = (*Discord)(nil)

type Discord struct {
	cfg        DiscordConfig
	httpClient *http.Client
}

func NewDiscord(cfg DiscordConfig, httpClient *http.Client) *Discord {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Discord{cfg: cfg, httpClient: httpClient}
}

func (d *Discord) AuthURL(state string) string {
	q := url.Values{}
	q.Set("client_id", d.cfg.ClientID)
	q.Set("redirect_uri", d.cfg.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "identify")
	q.Set("state", state)
	return d.cfg.BaseURL + "/oauth2/authorize?" + q.Encode()
}

func (d *Discord) Exchange(ctx context.Context, code string) (Identity, error) {
	token, err := d.exchangeCode(ctx, code)
	if err != nil {
		return Identity{}, fmt.Errorf("exchange discord code: %w", err)
	}
	user, err := d.fetchUser(ctx, token)
	if err != nil {
		return Identity{}, fmt.Errorf("fetch discord user: %w", err)
	}
	return Identity{
		ProviderUserID: user.ID,
		Username:       user.Username,
		AvatarURL:      avatarURL(user),
	}, nil
}

func (d *Discord) exchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", d.cfg.RedirectURL)
	form.Set("client_id", d.cfg.ClientID)
	form.Set("client_secret", d.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.cfg.BaseURL+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := d.do(req, &body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("token response has no access_token")
	}
	return body.AccessToken, nil
}

type discordUser struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Avatar   *string `json:"avatar"`
}

func (d *Discord) fetchUser(ctx context.Context, token string) (discordUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.cfg.BaseURL+"/users/@me", nil)
	if err != nil {
		return discordUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var user discordUser
	if err := d.do(req, &user); err != nil {
		return discordUser{}, err
	}
	return user, nil
}

func (d *Discord) do(req *http.Request, out any) error {
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Discord's error bodies are short JSON ({"error": "invalid_grant", ...}); keep enough to diagnose without dumping arbitrary payloads into logs.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: unexpected status %d: %s", req.Method, req.URL.Path, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", req.URL.Path, err)
	}
	return nil
}

func avatarURL(user discordUser) string {
	if user.Avatar == nil || *user.Avatar == "" {
		return ""
	}
	return fmt.Sprintf("%s/avatars/%s/%s.png", discordCDN, user.ID, *user.Avatar)
}
