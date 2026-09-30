package youtube

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

const (
	APIBaseURL = "https://www.googleapis.com"

	maxResultsPerPage = 50
)

type Config struct {
	APIKey  string
	BaseURL string
}

type Video struct {
	ID    string
	Title string
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

// PlaylistVideos returns the playlist's embeddable videos in playlist order.
// Private and deleted entries are dropped because their iframes only show an error.
func (c *Client) PlaylistVideos(ctx context.Context, playlistID string) ([]Video, error) {
	videos := []Video{}
	pageToken := ""
	for {
		page, err := c.fetchPage(ctx, playlistID, pageToken)
		if err != nil {
			return nil, fmt.Errorf("fetch youtube playlist items: %w", err)
		}
		for _, item := range page.Items {
			if item.Status.PrivacyStatus != "public" && item.Status.PrivacyStatus != "unlisted" {
				continue
			}
			videos = append(videos, Video{ID: item.Snippet.ResourceID.VideoID, Title: item.Snippet.Title})
		}
		if page.NextPageToken == "" {
			return videos, nil
		}
		pageToken = page.NextPageToken
	}
}

type playlistItemsPage struct {
	Items []struct {
		Snippet struct {
			Title      string `json:"title"`
			ResourceID struct {
				VideoID string `json:"videoId"`
			} `json:"resourceId"`
		} `json:"snippet"`
		Status struct {
			PrivacyStatus string `json:"privacyStatus"`
		} `json:"status"`
	} `json:"items"`
	NextPageToken string `json:"nextPageToken"`
}

func (c *Client) fetchPage(ctx context.Context, playlistID, pageToken string) (playlistItemsPage, error) {
	q := url.Values{}
	q.Set("part", "snippet,status")
	q.Set("playlistId", playlistID)
	q.Set("maxResults", fmt.Sprint(maxResultsPerPage))
	q.Set("key", c.cfg.APIKey)
	if pageToken != "" {
		q.Set("pageToken", pageToken)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/youtube/v3/playlistItems?"+q.Encode(), nil)
	if err != nil {
		return playlistItemsPage{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// url.Error embeds the full URL, which carries the API key.
		return playlistItemsPage{}, fmt.Errorf("GET %s: %w", req.URL.Path, urlErrorCause(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return playlistItemsPage{}, fmt.Errorf("GET %s: unexpected status %d: %s", req.URL.Path, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	var page playlistItemsPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return playlistItemsPage{}, fmt.Errorf("decode %s response: %w", req.URL.Path, err)
	}
	return page, nil
}

func urlErrorCause(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
