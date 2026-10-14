package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	APIBaseURL = "https://www.googleapis.com"

	maxResultsPerPage = 50

	// A live stream is one of the channel's uploads, but not always the newest:
	// Shorts posted while a long stream runs push it down the list. Checking more
	// uploads costs nothing on playlistItems.list and little on the batched videos.list.
	recentUploadsToCheck = 10
)

type Config struct {
	APIKey  string
	BaseURL string
}

type Video struct {
	ID    string
	Title string
}

type LiveStream struct {
	ChannelID    string
	VideoID      string
	Title        string
	ViewerCount  int
	ThumbnailURL string
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
	if pageToken != "" {
		q.Set("pageToken", pageToken)
	}

	var page playlistItemsPage
	if err := c.get(ctx, "/youtube/v3/playlistItems", q, &page); err != nil {
		return playlistItemsPage{}, err
	}
	return page, nil
}

// LiveStreams returns one stream per channel that is live now; channels YouTube
// does not report as live are simply absent. YouTube has no game category, so a
// live stream in any game counts.
func (c *Client) LiveStreams(ctx context.Context, channelIDs []string) ([]LiveStream, error) {
	videoIDs := []string{}
	for _, channelID := range channelIDs {
		uploads, err := c.recentUploads(ctx, channelID)
		if err != nil {
			return nil, fmt.Errorf("fetch recent uploads of %s: %w", channelID, err)
		}
		videoIDs = append(videoIDs, uploads...)
	}

	byChannel := map[string]LiveStream{}
	for start := 0; start < len(videoIDs); start += maxResultsPerPage {
		end := min(start+maxResultsPerPage, len(videoIDs))
		live, err := c.liveVideos(ctx, videoIDs[start:end])
		if err != nil {
			return nil, fmt.Errorf("fetch youtube videos: %w", err)
		}
		for _, stream := range live {
			if current, ok := byChannel[stream.ChannelID]; !ok || stream.ViewerCount > current.ViewerCount {
				byChannel[stream.ChannelID] = stream
			}
		}
	}

	streams := make([]LiveStream, 0, len(byChannel))
	for _, channelID := range channelIDs {
		if stream, ok := byChannel[channelID]; ok {
			streams = append(streams, stream)
		}
	}
	return streams, nil
}

// Avatars returns each known channel's profile image URL keyed by channel ID;
// channels YouTube does not know are absent.
func (c *Client) Avatars(ctx context.Context, channelIDs []string) (map[string]string, error) {
	avatars := make(map[string]string, len(channelIDs))
	for start := 0; start < len(channelIDs); start += maxResultsPerPage {
		end := min(start+maxResultsPerPage, len(channelIDs))
		q := url.Values{}
		q.Set("part", "snippet")
		q.Set("id", strings.Join(channelIDs[start:end], ","))
		q.Set("maxResults", fmt.Sprint(maxResultsPerPage))

		var body struct {
			Items []struct {
				ID      string `json:"id"`
				Snippet struct {
					Thumbnails thumbnails `json:"thumbnails"`
				} `json:"snippet"`
			} `json:"items"`
		}
		if err := c.get(ctx, "/youtube/v3/channels", q, &body); err != nil {
			return nil, fmt.Errorf("fetch youtube channels: %w", err)
		}
		for _, item := range body.Items {
			avatars[item.ID] = item.Snippet.Thumbnails.Medium.URL
		}
	}
	return avatars, nil
}

// Every channel's uploads playlist ID is its channel ID with the UC prefix swapped
// for UU, which saves a channels.list call per run.
func uploadsPlaylistID(channelID string) string {
	return "UU" + strings.TrimPrefix(channelID, "UC")
}

func (c *Client) recentUploads(ctx context.Context, channelID string) ([]string, error) {
	q := url.Values{}
	q.Set("part", "contentDetails")
	q.Set("playlistId", uploadsPlaylistID(channelID))
	q.Set("maxResults", fmt.Sprint(recentUploadsToCheck))

	var body struct {
		Items []struct {
			ContentDetails struct {
				VideoID string `json:"videoId"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := c.get(ctx, "/youtube/v3/playlistItems", q, &body); err != nil {
		// A channel with no uploads, or a mistyped channel ID, has no uploads playlist.
		// Skipping it keeps one bad row from stopping every other channel's update.
		var status *statusError
		if errors.As(err, &status) && status.code == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	ids := make([]string, 0, len(body.Items))
	for _, item := range body.Items {
		ids = append(ids, item.ContentDetails.VideoID)
	}
	return ids, nil
}

type thumbnails struct {
	Medium struct {
		URL string `json:"url"`
	} `json:"medium"`
}

func (c *Client) liveVideos(ctx context.Context, videoIDs []string) ([]LiveStream, error) {
	q := url.Values{}
	q.Set("part", "snippet,liveStreamingDetails")
	q.Set("id", strings.Join(videoIDs, ","))
	q.Set("maxResults", fmt.Sprint(maxResultsPerPage))

	var body struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				ChannelID            string     `json:"channelId"`
				Title                string     `json:"title"`
				LiveBroadcastContent string     `json:"liveBroadcastContent"`
				Thumbnails           thumbnails `json:"thumbnails"`
			} `json:"snippet"`
			LiveStreamingDetails struct {
				// YouTube sends the count as a string, and omits it when the owner hides it.
				ConcurrentViewers string `json:"concurrentViewers"`
			} `json:"liveStreamingDetails"`
		} `json:"items"`
	}
	if err := c.get(ctx, "/youtube/v3/videos", q, &body); err != nil {
		return nil, err
	}

	live := []LiveStream{}
	for _, item := range body.Items {
		if item.Snippet.LiveBroadcastContent != "live" {
			continue
		}
		viewers, _ := strconv.Atoi(item.LiveStreamingDetails.ConcurrentViewers)
		live = append(live, LiveStream{
			ChannelID:    item.Snippet.ChannelID,
			VideoID:      item.ID,
			Title:        item.Snippet.Title,
			ViewerCount:  viewers,
			ThumbnailURL: item.Snippet.Thumbnails.Medium.URL,
		})
	}
	return live, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	q.Set("key", c.cfg.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// url.Error embeds the full URL, which carries the API key.
		return fmt.Errorf("GET %s: %w", path, urlErrorCause(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &statusError{path: path, code: resp.StatusCode, body: strings.TrimSpace(string(snippet))}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}
	return nil
}

type statusError struct {
	path string
	code int
	body string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("GET %s: unexpected status %d: %s", e.path, e.code, e.body)
}

func urlErrorCause(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
