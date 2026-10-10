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

const BattleNetAPIBaseURL = "https://oauth.battle.net"

type BattleNetConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	BaseURL      string
}

var _ Provider = (*BattleNet)(nil)

type BattleNet struct {
	cfg        BattleNetConfig
	httpClient *http.Client
}

func NewBattleNet(cfg BattleNetConfig, httpClient *http.Client) *BattleNet {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &BattleNet{cfg: cfg, httpClient: httpClient}
}

func (b *BattleNet) AuthURL(state string) string {
	q := url.Values{}
	q.Set("client_id", b.cfg.ClientID)
	q.Set("redirect_uri", b.cfg.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "wow.profile")
	q.Set("state", state)
	return b.cfg.BaseURL + "/authorize?" + q.Encode()
}

func (b *BattleNet) Exchange(ctx context.Context, code string) (Identity, error) {
	token, err := b.exchangeCode(ctx, code)
	if err != nil {
		return Identity{}, fmt.Errorf("exchange battle.net code: %w", err)
	}
	user, err := b.fetchUser(ctx, token)
	if err != nil {
		return Identity{}, fmt.Errorf("fetch battle.net user: %w", err)
	}
	return Identity{ProviderUserID: fmt.Sprint(user.ID), Username: user.BattleTag}, nil
}

func (b *BattleNet) exchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", b.cfg.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.BaseURL+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(b.cfg.ClientID, b.cfg.ClientSecret)

	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := b.do(req, &body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("token response has no access_token")
	}
	return body.AccessToken, nil
}

type battleNetUser struct {
	ID        int64  `json:"id"`
	BattleTag string `json:"battletag"`
}

func (b *BattleNet) fetchUser(ctx context.Context, token string) (battleNetUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.cfg.BaseURL+"/userinfo", nil)
	if err != nil {
		return battleNetUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var user battleNetUser
	if err := b.do(req, &user); err != nil {
		return battleNetUser{}, err
	}
	if user.ID == 0 {
		return battleNetUser{}, fmt.Errorf("userinfo response has no id")
	}
	return user, nil
}

func (b *BattleNet) do(req *http.Request, out any) error {
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: unexpected status %d: %s", req.Method, req.URL.Path, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", req.URL.Path, err)
	}
	return nil
}
