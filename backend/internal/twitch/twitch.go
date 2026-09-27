package twitch

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
	APIBaseURL  = "https://api.twitch.tv"
	AuthBaseURL = "https://id.twitch.tv"

	// Retail World of Warcraft, looked up once via Get Games. Filtering on it
	// keeps a member streaming another game from showing as "live" in the guild.
	WorldOfWarcraftGameID = "18122"

	maxLoginsPerRequest = 100

	thumbnailSize = "440x248"
)

type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	AuthURL      string
}

type LiveStream struct {
	Login        string
	GameName     string
	Title        string
	ViewerCount  int
	ThumbnailURL string
}

// A Client is meant to live for one job run: it keeps the app token it fetched
// so LiveStreams and Avatars share one, and is not safe for concurrent use.
type Client struct {
	cfg        Config
	httpClient *http.Client
	token      string
}

func NewClient(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{cfg: cfg, httpClient: httpClient}
}

// LiveStreams returns only the channels that are live in the WoW category;
// channels Twitch does not report are simply absent.
func (c *Client) LiveStreams(ctx context.Context, logins []string) ([]LiveStream, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch twitch app token: %w", err)
	}

	live := []LiveStream{}
	for start := 0; start < len(logins); start += maxLoginsPerRequest {
		end := min(start+maxLoginsPerRequest, len(logins))
		batch, err := c.fetchStreams(ctx, token, logins[start:end])
		if err != nil {
			return nil, fmt.Errorf("fetch twitch streams: %w", err)
		}
		live = append(live, batch...)
	}
	return live, nil
}

// Avatars returns each known user's profile image URL keyed by lowercase login;
// logins Twitch does not know are absent.
func (c *Client) Avatars(ctx context.Context, logins []string) (map[string]string, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch twitch app token: %w", err)
	}

	avatars := make(map[string]string, len(logins))
	for start := 0; start < len(logins); start += maxLoginsPerRequest {
		end := min(start+maxLoginsPerRequest, len(logins))
		if err := c.fetchUsers(ctx, token, logins[start:end], avatars); err != nil {
			return nil, fmt.Errorf("fetch twitch users: %w", err)
		}
	}
	return avatars, nil
}

func (c *Client) fetchUsers(ctx context.Context, token string, logins []string, into map[string]string) error {
	q := url.Values{}
	for _, login := range logins {
		q.Add("login", login)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/helix/users?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Client-Id", c.cfg.ClientID)

	var body struct {
		Data []struct {
			Login           string `json:"login"`
			ProfileImageURL string `json:"profile_image_url"`
		} `json:"data"`
	}
	if err := c.do(req, &body); err != nil {
		return err
	}
	for _, u := range body.Data {
		into[strings.ToLower(u.Login)] = u.ProfileImageURL
	}
	return nil
}

func (c *Client) appToken(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}

	form := url.Values{}
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.AuthURL+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do(req, &body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("token response has no access_token")
	}
	c.token = body.AccessToken
	return c.token, nil
}

type helixStream struct {
	UserLogin    string `json:"user_login"`
	GameName     string `json:"game_name"`
	Title        string `json:"title"`
	ViewerCount  int    `json:"viewer_count"`
	ThumbnailURL string `json:"thumbnail_url"`
}

func (c *Client) fetchStreams(ctx context.Context, token string, logins []string) ([]LiveStream, error) {
	q := url.Values{}
	for _, login := range logins {
		q.Add("user_login", login)
	}
	q.Set("game_id", WorldOfWarcraftGameID)
	q.Set("first", fmt.Sprint(maxLoginsPerRequest))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/helix/streams?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Client-Id", c.cfg.ClientID)

	var body struct {
		Data []helixStream `json:"data"`
	}
	if err := c.do(req, &body); err != nil {
		return nil, err
	}

	live := make([]LiveStream, 0, len(body.Data))
	for _, s := range body.Data {
		live = append(live, LiveStream{
			Login:        strings.ToLower(s.UserLogin),
			GameName:     s.GameName,
			Title:        s.Title,
			ViewerCount:  s.ViewerCount,
			ThumbnailURL: strings.NewReplacer("{width}x{height}", thumbnailSize).Replace(s.ThumbnailURL),
		})
	}
	return live, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
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
