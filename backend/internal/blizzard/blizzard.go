package blizzard

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

// Classic race ids plus the Skyborne ids seen in a real Forever roster. High Order's id is
// not known yet; a member with an unmapped race keeps RaceID and an empty Race.
var raceNames = map[int]string{
	1: "Human", 2: "Orc", 3: "Dwarf", 4: "Night Elf",
	5: "Undead", 6: "Tauren", 7: "Gnome", 8: "Troll",
	96: "Skyborne (Windshaper)",
}

type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	AuthURL      string
	// Namespace is the Battlenet-Namespace for the game version, e.g. profile-us for
	// retail or profile-forever-us for Forever.
	Namespace string
	Locale    string
}

// RosterMember is one guild member. Class is empty when the class id is not one the
// site supports; ClassID is kept so a caller can say which one it skipped.
type RosterMember struct {
	Name    string
	Surname string
	Level   int
	Rank    int
	ClassID int
	Class   string
	RaceID  int
	Race    string
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

// GuildRoster returns every member of the guild. Forever has no realms, so a character
// is identified by its name and surname. The slug is the lowercase hyphenated form
// Blizzard uses in URLs.
func (c *Client) GuildRoster(ctx context.Context, guildSlug string) ([]RosterMember, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch blizzard app token: %w", err)
	}

	q := url.Values{}
	q.Set("namespace", c.cfg.Namespace)
	q.Set("locale", c.cfg.Locale)
	endpoint := fmt.Sprintf("%s/data/wow/guild/%s/roster?%s",
		c.cfg.BaseURL, url.PathEscape(guildSlug), q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var body struct {
		Members []struct {
			Character struct {
				Name          string `json:"name"`
				Level         int    `json:"level"`
				Surname       string `json:"surname"`
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
			Name:    m.Character.Name,
			Surname: m.Character.Surname,
			Level:   m.Character.Level,
			Rank:    m.Rank,
			ClassID: m.Character.PlayableClass.ID,
			Class:   classNames[m.Character.PlayableClass.ID],
			RaceID:  m.Character.PlayableRace.ID,
			Race:    raceNames[m.Character.PlayableRace.ID],
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
		err := fmt.Errorf("%s %s: unexpected status %d: %s", req.Method, req.URL.Path, resp.StatusCode, strings.TrimSpace(string(snippet)))
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return err
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", req.URL.Path, err)
	}
	return nil
}

// ErrNotFound is wrapped by every call Blizzard answers with 404.
var ErrNotFound = errors.New("not found")

// CharacterGender returns "male" or "female" from the character's profile summary.
// characterSlug is name-surname, lowercase. A character Blizzard has no profile for
// yields an error wrapping ErrNotFound.
func (c *Client) CharacterGender(ctx context.Context, characterSlug string) (string, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return "", fmt.Errorf("fetch blizzard app token: %w", err)
	}

	q := url.Values{}
	q.Set("namespace", c.cfg.Namespace)
	q.Set("locale", c.cfg.Locale)
	endpoint := fmt.Sprintf("%s/profile/wow/character/%s?%s",
		c.cfg.BaseURL, url.PathEscape(characterSlug), q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var body struct {
		Gender struct {
			Type string `json:"type"`
		} `json:"gender"`
	}
	if err := c.do(req, &body); err != nil {
		return "", fmt.Errorf("fetch profile of %s: %w", characterSlug, err)
	}

	switch gender := strings.ToLower(body.Gender.Type); gender {
	case "male", "female":
		return gender, nil
	default:
		return "", fmt.Errorf("profile of %s has unknown gender %q", characterSlug, body.Gender.Type)
	}
}

// Recipe is one thing a character can craft. The id is Blizzard's own; it is not the
// spell id Wowhead uses.
type Recipe struct {
	ID   int
	Name string
}

// CharacterProfession is one profession a character knows. SkillPoints comes from the
// highest tier, which on Forever is its only tier; Recipes spans all tiers.
type CharacterProfession struct {
	Name        string
	SkillPoints int
	Recipes     []Recipe
}

// CharacterProfessions returns a character's primary then secondary professions.
// characterSlug is name-surname, lowercase. A character Blizzard has no profile for
// yields an error wrapping ErrNotFound.
func (c *Client) CharacterProfessions(ctx context.Context, characterSlug string) ([]CharacterProfession, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch blizzard app token: %w", err)
	}

	q := url.Values{}
	q.Set("namespace", c.cfg.Namespace)
	q.Set("locale", c.cfg.Locale)
	endpoint := fmt.Sprintf("%s/profile/wow/character/%s/professions?%s",
		c.cfg.BaseURL, url.PathEscape(characterSlug), q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	type entry struct {
		Profession struct {
			Name string `json:"name"`
		} `json:"profession"`
		Tiers []struct {
			SkillPoints  int `json:"skill_points"`
			KnownRecipes []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"known_recipes"`
		} `json:"tiers"`
	}
	var body struct {
		Primaries   []entry `json:"primaries"`
		Secondaries []entry `json:"secondaries"`
	}
	if err := c.do(req, &body); err != nil {
		return nil, fmt.Errorf("fetch professions of %s: %w", characterSlug, err)
	}

	var professions []CharacterProfession
	for _, e := range append(body.Primaries, body.Secondaries...) {
		profession := CharacterProfession{Name: e.Profession.Name}
		for _, tier := range e.Tiers {
			profession.SkillPoints = max(profession.SkillPoints, tier.SkillPoints)
			for _, r := range tier.KnownRecipes {
				profession.Recipes = append(profession.Recipes, Recipe{ID: r.ID, Name: r.Name})
			}
		}
		professions = append(professions, profession)
	}
	return professions, nil
}
