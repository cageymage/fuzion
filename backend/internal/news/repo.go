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

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ListPosts(ctx context.Context) ([]Post, error) {
	const query = `
		SELECT id, title, excerpt, category, image_url, author_name, published_at
		FROM news_posts
		ORDER BY published_at DESC`

	posts := []Post{}
	if err := r.db.SelectContext(ctx, &posts, query); err != nil {
		return nil, fmt.Errorf("select news posts: %w", err)
	}
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	for i := range posts {
		posts[i].PublishedAt = posts[i].PublishedAt.UTC()
	}
	return posts, nil
}
