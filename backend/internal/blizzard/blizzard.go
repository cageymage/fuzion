package blizzard

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
	AuthBaseURL = "https://oauth.battle.net"

	DefaultLocale = "en_US"
)

// APIBaseURL is the regional host for game data and profile calls.
func APIBaseURL(region string) string {
	return "https://" + region + ".api.blizzard.com"
}

// The guild roster response carries only a numeric class id, no name. These are the
// Classic ids, which is all Forever has: Death Knight, Monk, Demon Hunter and Evoker
// are retail-only, so they are deliberately absent.
var classNames = map[int]string{
	1: "Warrior", 2: "Paladin", 3: "Hunter", 4: "Rogue", 5: "Priest",
	7: "Shaman", 8: "Mage", 9: "Warlock", 11: "Druid",
}

// Classic race ids. Forever's Skyborne races are not here because their ids are not
// known yet; a member with an unmapped race keeps RaceID and an empty Race.
var raceNames = map[int]string{
	1: "Human", 2: "Orc", 3: "Dwarf", 4: "Night Elf",
	5: "Undead", 6: "Tauren", 7: "Gnome", 8: "Troll",
}

type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	AuthURL      string
	// Namespace is the Battlenet-Namespace for the game version, e.g. profile-us for
	// retail or profile-classic1x-us for Classic Era. Forever's value is not known yet.
	Namespace string
	Locale    string
}

// RosterMember is one guild member. Class is empty when the class id is not one the
// site supports; ClassID is kept so a caller can say which one it skipped.
type RosterMember struct {
	Name      string
	RealmSlug string
	Level     int
	Rank      int
	ClassID   int
	Class     string
	RaceID    int
	Race      string
}

// A Client is meant to live for one job run: it keeps the app token it fetched and is
// not safe for concurrent use.
type Client struct {
	cfg        Config
	httpClient *http.Client
	token      string
}

func NewClient(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if cfg.Locale == "" {
		cfg.Locale = DefaultLocale
	}
	return &Client{cfg: cfg, httpClient: httpClient}
}

// GuildRoster returns every member of the guild. Each member carries its own realm
// slug because retail guilds can span realms. Slugs are the lowercase hyphenated
// forms Blizzard uses in URLs.
func (c *Client) GuildRoster(ctx context.Context, realmSlug, guildSlug string) ([]RosterMember, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch blizzard app token: %w", err)
	}

	q := url.Values{}
	q.Set("namespace", c.cfg.Namespace)
	q.Set("locale", c.cfg.Locale)
	endpoint := fmt.Sprintf("%s/data/wow/guild/%s/%s/roster?%s",
		c.cfg.BaseURL, url.PathEscape(realmSlug), url.PathEscape(guildSlug), q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var body struct {
		Members []struct {
			Character struct {
				Name  string `json:"name"`
				Level int    `json:"level"`
				Realm struct {
					Slug string `json:"slug"`
				} `json:"realm"`
				PlayableClass struct {
					ID int `json:"id"`
				} `json:"playable_class"`
				PlayableRace struct {
					ID int `json:"id"`
				} `json:"playable_race"`
			} `json:"character"`
			Rank int `json:"rank"`
		} `json:"members"`
	}
	if err := c.do(req, &body); err != nil {
		return nil, fmt.Errorf("fetch guild roster: %w", err)
	}

	members := make([]RosterMember, 0, len(body.Members))
	for _, m := range body.Members {
		members = append(members, RosterMember{
			Name:      m.Character.Name,
			RealmSlug: m.Character.Realm.Slug,
			Level:     m.Character.Level,
			Rank:      m.Rank,
			ClassID:   m.Character.PlayableClass.ID,
			Class:     classNames[m.Character.PlayableClass.ID],
			RaceID:    m.Character.PlayableRace.ID,
			Race:      raceNames[m.Character.PlayableRace.ID],
		})
	}
	return members, nil
}

func (c *Client) appToken(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.AuthURL+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)

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
