package images

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("image not found")

type NewImage struct {
	ContentType string
	Width       int
	Height      int
	FullBytes   []byte
	ThumbBytes  []byte
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(ctx context.Context, image NewImage) (uuid.UUID, error) {
	const query = `
		INSERT INTO images (content_type, width, height, full_bytes, thumb_bytes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id uuid.UUID
	err := r.db.GetContext(ctx, &id, query, image.ContentType, image.Width, image.Height, image.FullBytes, image.ThumbBytes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert image: %w", err)
	}
	return id, nil
}

func (r *Repo) Full(ctx context.Context, id uuid.UUID) ([]byte, error) {
	const query = `SELECT full_bytes FROM images WHERE id = $1`
	return r.bytes(ctx, query, id)
}

func (r *Repo) Thumb(ctx context.Context, id uuid.UUID) ([]byte, error) {
	const query = `SELECT thumb_bytes FROM images WHERE id = $1`
	return r.bytes(ctx, query, id)
}

func (r *Repo) bytes(ctx context.Context, query string, id uuid.UUID) ([]byte, error) {
	var data []byte
	err := r.db.GetContext(ctx, &data, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select image %s: %w", id, err)
	}
	return data, nil
}
