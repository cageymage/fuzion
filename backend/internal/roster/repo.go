package roster

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Character struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	Realm     string    `db:"realm"      json:"realm"`
	Class     string    `db:"class"      json:"class"`
	Spec      *string   `db:"spec"       json:"spec"`
	Role      string    `db:"role"       json:"role"`
	IsMain    bool      `db:"is_main"    json:"isMain"`
	RaidTeam  *string   `db:"raid_team"  json:"raidTeam"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List(ctx context.Context) ([]Character, error) {
	const query = `
		SELECT id, name, realm, class, spec, role, is_main, raid_team, created_at
		FROM characters
		ORDER BY is_main DESC, name ASC`

	characters := []Character{}
	if err := r.db.SelectContext(ctx, &characters, query); err != nil {
		return nil, fmt.Errorf("select characters: %w", err)
	}
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	for i := range characters {
		characters[i].CreatedAt = characters[i].CreatedAt.UTC()
	}
	return characters, nil
}
