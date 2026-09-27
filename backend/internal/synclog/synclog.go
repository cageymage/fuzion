package synclog

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Record(ctx context.Context, source, status, message string) error {
	const query = `INSERT INTO sync_log (source, status, message) VALUES ($1, $2, $3)`

	if _, err := r.db.ExecContext(ctx, query, source, status, message); err != nil {
		return fmt.Errorf("insert %s sync log: %w", source, err)
	}
	return nil
}
