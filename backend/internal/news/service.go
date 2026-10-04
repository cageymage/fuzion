package news

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/clock"
	"github.com/cageymage/fuzion/backend/internal/discord"
)

const (
	maxTitleLength   = 120
	maxExcerptLength = 300
	maxBodyLength    = 50000
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

// Notifier is the announcements channel; a nil Notifier means cross-posting is disabled.
type Notifier interface {
	Send(ctx context.Context, msg discord.Message) error
}

type CreateRequest struct {
	Title    string `json:"title"`
	Excerpt  string `json:"excerpt"`
	Category string `json:"category"`
	Body     string `json:"body"`
	Pinned   bool   `json:"pinned"`
}

type UpdateRequest struct {
	Title    *string `json:"title"`
	Excerpt  *string `json:"excerpt"`
	Category *string `json:"category"`
	Body     *string `json:"body"`
	Pinned   *bool   `json:"pinned"`
}

type Service struct {
	repo        *Repo
	clock       clock.Clock
	notifier    Notifier
	siteBaseURL string
}

func NewService(repo *Repo, clock clock.Clock, notifier Notifier, siteBaseURL string) *Service {
	return &Service{repo: repo, clock: clock, notifier: notifier, siteBaseURL: strings.TrimRight(siteBaseURL, "/")}
}

type Page struct {
	Posts []Post
	Total int
}

func (s *Service) ListPosts(ctx context.Context, params ListParams) (Page, error) {
	posts, err := s.repo.ListPosts(ctx, params)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	total, err := s.repo.CountPosts(ctx, params.Category)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	return Page{Posts: posts, Total: total}, nil
}

func (s *Service) GetPublished(ctx context.Context, id string) (Post, error) {
	postID, err := parseID(id)
	if err != nil {
		return Post{}, err
	}
	post, err := s.repo.GetPublished(ctx, postID)
	if err != nil {
		return Post{}, fmt.Errorf("get post: %w", err)
	}
	return post, nil
}

func (s *Service) ListDrafts(ctx context.Context) ([]Post, error) {
	posts, err := s.repo.ListDrafts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list drafts: %w", err)
	}
	return posts, nil
}

func (s *Service) Create(ctx context.Context, author uuid.UUID, authorName string, req CreateRequest) (Post, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Excerpt = strings.TrimSpace(req.Excerpt)
	if err := validate(&req.Title, &req.Excerpt, &req.Category, &req.Body); err != nil {
		return Post{}, err
	}

	post, err := s.repo.Create(ctx, NewPost{
		Title:        req.Title,
		Excerpt:      req.Excerpt,
		Category:     req.Category,
		Body:         req.Body,
		Pinned:       req.Pinned,
		AuthorUserID: author,
		AuthorName:   authorName,
	})
	if err != nil {
		return Post{}, fmt.Errorf("create post: %w", err)
	}
	return post, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Post, error) {
	postID, err := parseID(id)
	if err != nil {
		return Post{}, err
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		req.Title = &trimmed
	}
	if req.Excerpt != nil {
		trimmed := strings.TrimSpace(*req.Excerpt)
		req.Excerpt = &trimmed
	}
	if err := validate(req.Title, req.Excerpt, req.Category, req.Body); err != nil {
		return Post{}, err
	}

	post, err := s.repo.Update(ctx, postID, PostChanges(req))
	if err != nil {
		return Post{}, fmt.Errorf("update post: %w", err)
	}
	return post, nil
}

func (s *Service) Publish(ctx context.Context, id string) (Post, error) {
	postID, err := parseID(id)
	if err != nil {
		return Post{}, err
	}
	post, firstPublish, err := s.repo.Publish(ctx, postID, s.clock.Now())
	if err != nil {
		return Post{}, fmt.Errorf("publish post: %w", err)
	}
	if firstPublish {
		s.announce(ctx, post)
	}
	return post, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	postID, err := parseID(id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, postID); err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	return nil
}

// The post is already published, so a failed cross-post must not fail the request.
func (s *Service) announce(ctx context.Context, post Post) {
	if s.notifier == nil {
		return
	}
	embed := discord.Embed{
		Title:       post.Title,
		URL:         s.siteBaseURL + "/news/" + post.ID.String(),
		Description: post.Excerpt,
		Fields:      []discord.EmbedField{{Name: "Category", Value: post.Category, Inline: true}},
	}
	if src := firstImageURL(post.Body); src != "" {
		if uploadedImagePath.MatchString(src) {
			src += "/thumb"
		}
		if strings.HasPrefix(src, "/") && !strings.HasPrefix(src, "//") {
			src = s.siteBaseURL + src
		}
		embed.Image = &discord.EmbedImage{URL: src}
	}
	msg := discord.Message{Embeds: []discord.Embed{embed}}
	if err := s.notifier.Send(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "announce published news post", "postID", post.ID, "error", err)
	}
}

var uploadedImagePath = regexp.MustCompile(`^/api/images/[0-9a-fA-F-]{36}$`)

var markdownImage =regexp.MustCompile(`!\[[^\]]*\]\(\s*<?([^\s)>]+)>?(?:\s+(?:"[^"]*"|'[^']*'))?\s*\)`)

func firstImageURL(body string) string {
	match := markdownImage.FindStringSubmatch(body)
	if match == nil {
		return ""
	}
	return match[1]
}

func parseID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, &ValidationError{Field: "id", Problem: "must be a valid UUID"}
	}
	return parsed, nil
}

// Nil arguments are skipped so a partial update validates only what it changes.
func validate(title, excerpt, category, body *string) error {
	if title != nil {
		if n := utf8.RuneCountInString(*title); n < 1 || n > maxTitleLength {
			return &ValidationError{Field: "title", Problem: fmt.Sprintf("must be 1 to %d characters", maxTitleLength)}
		}
	}
	if excerpt != nil && utf8.RuneCountInString(*excerpt) > maxExcerptLength {
		return &ValidationError{Field: "excerpt", Problem: fmt.Sprintf("must be at most %d characters", maxExcerptLength)}
	}
	if category != nil && !slices.Contains(categories, *category) {
		return &ValidationError{Field: "category", Problem: "must be one of " + strings.Join(categories, ", ")}
	}
	if body != nil && utf8.RuneCountInString(*body) > maxBodyLength {
		return &ValidationError{Field: "body", Problem: fmt.Sprintf("must be at most %d characters", maxBodyLength)}
	}
	return nil
}
