package news

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Post struct {
	ID          uuid.UUID `db:"id"           json:"id"`
	Title       string    `db:"title"        json:"title"`
	Excerpt     string    `db:"excerpt"      json:"excerpt"`
	Category    string    `db:"category"     json:"category"`
	ImageURL    *string   `db:"image_url"    json:"imageUrl"`
	AuthorName  string    `db:"author_name"  json:"authorName"`
	PublishedAt time.Time `db:"published_at" json:"publishedAt"`
}

type ListParams struct {
	Limit    int
	Offset   int
	Category string
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ListPosts(ctx context.Context, params ListParams) ([]Post, error) {
	// An empty category means every category. id breaks ties so pages are stable
	// when two posts share a timestamp.
	const query = `
		SELECT id, title, excerpt, category, image_url, author_name, published_at
		FROM news_posts
		WHERE ($1::text = '' OR category = $1)
		ORDER BY published_at DESC, id
		LIMIT $2 OFFSET $3`

	posts := []Post{}
	if err := r.db.SelectContext(ctx, &posts, query, params.Category, params.Limit, params.Offset); err != nil {
		return nil, fmt.Errorf("select news posts: %w", err)
	}
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	for i := range posts {
		posts[i].PublishedAt = posts[i].PublishedAt.UTC()
	}
	return posts, nil
}

func (r *Repo) CountPosts(ctx context.Context, category string) (int, error) {
	const query = `SELECT COUNT(*) FROM news_posts WHERE ($1::text = '' OR category = $1)`

	var total int
	if err := r.db.GetContext(ctx, &total, query, category); err != nil {
		return 0, fmt.Errorf("count news posts: %w", err)
	}
	return total, nil
}
