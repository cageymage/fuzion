package news

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("news post not found")

type Post struct {
	ID           uuid.UUID  `db:"id"             json:"id"`
	Title        string     `db:"title"          json:"title"`
	Excerpt      string     `db:"excerpt"        json:"excerpt"`
	Category     string     `db:"category"       json:"category"`
	Body         string     `db:"body"           json:"body"`
	Pinned       bool       `db:"pinned"         json:"pinned"`
	ImageURL     *string    `db:"image_url"      json:"imageUrl"`
	AuthorName   string     `db:"author_name"    json:"authorName"`
	AuthorUserID *uuid.UUID `db:"author_user_id" json:"-"`
	PublishedAt  *time.Time `db:"published_at"   json:"publishedAt"`
	UpdatedAt    time.Time  `db:"updated_at"     json:"updatedAt"`
}

type ListParams struct {
	Limit    int
	Offset   int
	Category string
}

type NewPost struct {
	Title        string
	Excerpt      string
	Category     string
	Body         string
	Pinned       bool
	AuthorUserID uuid.UUID
	AuthorName   string
}

// A nil field is left unchanged.
type PostChanges struct {
	Title    *string
	Excerpt  *string
	Category *string
	Body     *string
	Pinned   *bool
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

const postColumns = `id, title, excerpt, category, body, pinned, image_url, author_name, author_user_id, published_at, updated_at`

func (p *Post) normalizeTimes() {
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	if p.PublishedAt != nil {
		utc := p.PublishedAt.UTC()
		p.PublishedAt = &utc
	}
	p.UpdatedAt = p.UpdatedAt.UTC()
}

func (r *Repo) ListPosts(ctx context.Context, params ListParams) ([]Post, error) {
	// An empty category means every category. id breaks ties so pages are stable
	// when two posts share a timestamp.
	const query = `
		SELECT ` + postColumns + `
		FROM news_posts
		WHERE published_at IS NOT NULL AND ($1::text = '' OR category = $1)
		ORDER BY pinned DESC, published_at DESC, id
		LIMIT $2 OFFSET $3`

	posts := []Post{}
	if err := r.db.SelectContext(ctx, &posts, query, params.Category, params.Limit, params.Offset); err != nil {
		return nil, fmt.Errorf("select news posts: %w", err)
	}
	for i := range posts {
		posts[i].normalizeTimes()
	}
	return posts, nil
}

func (r *Repo) CountPosts(ctx context.Context, category string) (int, error) {
	const query = `SELECT COUNT(*) FROM news_posts WHERE published_at IS NOT NULL AND ($1::text = '' OR category = $1)`

	var total int
	if err := r.db.GetContext(ctx, &total, query, category); err != nil {
		return 0, fmt.Errorf("count news posts: %w", err)
	}
	return total, nil
}

func (r *Repo) GetPublished(ctx context.Context, id uuid.UUID) (Post, error) {
	const query = `SELECT ` + postColumns + ` FROM news_posts WHERE id = $1 AND published_at IS NOT NULL`

	var post Post
	err := r.db.GetContext(ctx, &post, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, fmt.Errorf("select published news post %s: %w", id, err)
	}
	post.normalizeTimes()
	return post, nil
}

func (r *Repo) ListDrafts(ctx context.Context) ([]Post, error) {
	const query = `SELECT ` + postColumns + ` FROM news_posts WHERE published_at IS NULL ORDER BY updated_at DESC, id`

	posts := []Post{}
	if err := r.db.SelectContext(ctx, &posts, query); err != nil {
		return nil, fmt.Errorf("select news drafts: %w", err)
	}
	for i := range posts {
		posts[i].normalizeTimes()
	}
	return posts, nil
}

func (r *Repo) Create(ctx context.Context, post NewPost) (Post, error) {
	const query = `
		INSERT INTO news_posts (title, excerpt, category, body, pinned, author_user_id, author_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + postColumns

	var created Post
	err := r.db.GetContext(ctx, &created, query,
		post.Title, post.Excerpt, post.Category, post.Body, post.Pinned, post.AuthorUserID, post.AuthorName)
	if err != nil {
		return Post{}, fmt.Errorf("insert news post: %w", err)
	}
	created.normalizeTimes()
	return created, nil
}

func (r *Repo) Update(ctx context.Context, id uuid.UUID, changes PostChanges) (Post, error) {
	const query = `
		UPDATE news_posts SET
			title      = COALESCE($2, title),
			excerpt    = COALESCE($3, excerpt),
			category   = COALESCE($4, category),
			body       = COALESCE($5, body),
			pinned     = COALESCE($6, pinned),
			updated_at = now()
		WHERE id = $1
		RETURNING ` + postColumns

	var updated Post
	err := r.db.GetContext(ctx, &updated, query,
		id, changes.Title, changes.Excerpt, changes.Category, changes.Body, changes.Pinned)
	if errors.Is(err, sql.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, fmt.Errorf("update news post %s: %w", id, err)
	}
	updated.normalizeTimes()
	return updated, nil
}

// firstPublish is true only when this call moved the post out of draft.
func (r *Repo) Publish(ctx context.Context, id uuid.UUID, now time.Time) (post Post, firstPublish bool, err error) {
	const query = `
		WITH prior AS (
			SELECT published_at IS NULL AS was_draft FROM news_posts WHERE id = $1 FOR UPDATE
		)
		UPDATE news_posts SET
			published_at = COALESCE(published_at, $2),
			updated_at   = CASE WHEN published_at IS NULL THEN $2 ELSE updated_at END
		FROM prior
		WHERE news_posts.id = $1
		RETURNING ` + postColumns + `, prior.was_draft`

	var row struct {
		Post
		WasDraft bool `db:"was_draft"`
	}
	err = r.db.GetContext(ctx, &row, query, id, now)
	if errors.Is(err, sql.ErrNoRows) {
		return Post{}, false, ErrNotFound
	}
	if err != nil {
		return Post{}, false, fmt.Errorf("publish news post %s: %w", id, err)
	}
	row.Post.normalizeTimes()
	return row.Post, row.WasDraft, nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM news_posts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete news post %s: %w", id, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete news post %s: %w", id, err)
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}
