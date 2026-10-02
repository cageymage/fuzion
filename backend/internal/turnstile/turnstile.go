package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const VerifyBaseURL = "https://challenges.cloudflare.com"

// ErrInvalidToken means Cloudflare answered and rejected the token, as opposed
// to Cloudflare being unreachable or misbehaving.
var ErrInvalidToken = errors.New("turnstile token rejected")

type Config struct {
	SecretKey string
	BaseURL   string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{cfg: cfg, httpClient: httpClient}
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify checks token with Cloudflare. remoteIP is optional and may be empty.
func (c *Client) Verify(ctx context.Context, token, remoteIP string) error {
	form := url.Values{}
	form.Set("secret", c.cfg.SecretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	const path = "/turnstile/v0/siteverify"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build POST %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, urlErrorCause(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("POST %s: unexpected status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	var result siteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}
	if !result.Success {
		return fmt.Errorf("%w: %s", ErrInvalidToken, strings.Join(result.ErrorCodes, ", "))
	}
	return nil
}

func urlErrorCause(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
