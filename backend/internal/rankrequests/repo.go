package rankrequests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type Request struct {
	ID                     uuid.UUID  `db:"id"                       json:"id"`
	CharacterID            uuid.UUID  `db:"character_id"             json:"characterId"`
	CharacterName          string     `db:"character_name"           json:"characterName"`
	CharacterSecondaryName string     `db:"character_secondary_name" json:"characterSecondaryName"`
	CharacterClass         string     `db:"character_class"          json:"characterClass"`
	Direction              string     `db:"direction"                json:"direction"`
	Status                 string     `db:"status"                   json:"status"`
	Note                   string     `db:"note"                     json:"note"`
	ResolutionNote         string     `db:"resolution_note"          json:"resolutionNote"`
	RequestedByID          uuid.UUID  `db:"requested_by"             json:"-"`
	CharacterOwnerID       *uuid.UUID `db:"character_owner"          json:"-"`
	RequestedBy            string     `db:"requested_by_name"        json:"requestedBy"`
	CreatedAt              time.Time  `db:"created_at"               json:"createdAt"`
	ResolvedAt             *time.Time `db:"resolved_at"              json:"resolvedAt"`
	ResolvedBy             *string    `db:"resolved_by_name"         json:"resolvedBy"`
}

const selectRequests = `
	SELECT r.id, r.character_id, c.name AS character_name, c.secondary_name AS character_secondary_name,
	       c.class AS character_class, c.user_id AS character_owner, r.direction, r.status, r.note, r.resolution_note,
	       r.requested_by, requester.username AS requested_by_name, r.created_at, r.resolved_at,
	       resolver.username AS resolved_by_name
	FROM rank_change_requests r
	JOIN characters c ON c.id = r.character_id
	JOIN users requester ON requester.id = r.requested_by
	LEFT JOIN users resolver ON resolver.id = r.resolved_by`

// resolvedHistoryLimit keeps the officer page to recent history; older requests stay in the table.
const resolvedHistoryLimit = 20

const uniqueViolationCode = "23505"

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) characterOwner(ctx context.Context, id uuid.UUID) (*uuid.UUID, error) {
	var owner *uuid.UUID
	err := r.db.GetContext(ctx, &owner, `SELECT user_id FROM characters WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCharacterNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select character owner: %w", err)
	}
	return owner, nil
}

// Create files the request only while requestedBy still owns the character, and takes
// the direction from the character's main/alt state in the same statement, so a sync or
// an officer changing either between the service's check and the insert cannot slip
// past it. It returns ErrCharacterNotOwned when the owner changed in between.
func (r *Repo) Create(ctx context.Context, id, characterID, requestedBy uuid.UUID, note string, createdAt time.Time) (Request, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO rank_change_requests (id, character_id, requested_by, direction, note, created_at)
		SELECT $1, c.id, $3, CASE WHEN c.is_main THEN 'to_alt' ELSE 'to_main' END, $4, $5
		FROM characters c
		WHERE c.id = $2 AND c.user_id = $3`, id, characterID, requestedBy, note, createdAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return Request{}, ErrOpenRequestExists
	}
	if err != nil {
		return Request{}, fmt.Errorf("insert rank request: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Request{}, fmt.Errorf("count inserted rank requests: %w", err)
	}
	if n == 0 {
		return Request{}, ErrCharacterNotOwned
	}
	return r.Get(ctx, id)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (Request, error) {
	var req Request
	err := r.db.GetContext(ctx, &req, selectRequests+` WHERE r.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	if err != nil {
		return Request{}, fmt.Errorf("select rank request: %w", err)
	}
	return inUTC(req), nil
}

// ListLatestForOwner returns the newest request that was not cancelled for each
// character the user owns, newest first.
func (r *Repo) ListLatestForOwner(ctx context.Context, userID uuid.UUID) ([]Request, error) {
	query := selectRequests + `
		WHERE c.user_id = $1 AND r.status <> 'cancelled'
		  AND r.created_at = (
			SELECT max(created_at) FROM rank_change_requests
			WHERE character_id = r.character_id AND status <> 'cancelled'
		  )
		ORDER BY r.created_at DESC`

	requests := []Request{}
	if err := r.db.SelectContext(ctx, &requests, query, userID); err != nil {
		return nil, fmt.Errorf("select my rank requests: %w", err)
	}
	for i := range requests {
		requests[i] = inUTC(requests[i])
	}
	return requests, nil
}

// ListForOfficers returns every open request oldest first, then the most recently
// resolved ones. Cancelled requests never needed an officer, so they are left out.
func (r *Repo) ListForOfficers(ctx context.Context) ([]Request, error) {
	open := []Request{}
	if err := r.db.SelectContext(ctx, &open, selectRequests+` WHERE r.status = 'open' ORDER BY r.created_at ASC`); err != nil {
		return nil, fmt.Errorf("select open rank requests: %w", err)
	}
	var resolved []Request
	if err := r.db.SelectContext(ctx, &resolved, selectRequests+`
		WHERE r.status IN ('done', 'declined') ORDER BY r.resolved_at DESC LIMIT $1`, resolvedHistoryLimit); err != nil {
		return nil, fmt.Errorf("select resolved rank requests: %w", err)
	}
	requests := append(open, resolved...)
	for i := range requests {
		requests[i] = inUTC(requests[i])
	}
	return requests, nil
}

func (r *Repo) CountOpen(ctx context.Context) (int, error) {
	var n int
	if err := r.db.GetContext(ctx, &n, `SELECT count(*) FROM rank_change_requests WHERE status = 'open'`); err != nil {
		return 0, fmt.Errorf("count open rank requests: %w", err)
	}
	return n, nil
}

// Close moves an open request to status. It returns ErrNotOpen when the request
// exists but was already closed, so two officers acting at once cannot both win.
func (r *Repo) Close(ctx context.Context, id uuid.UUID, status, resolutionNote string, resolvedBy *uuid.UUID, resolvedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE rank_change_requests
		SET status = $2, resolution_note = $3, resolved_by = $4, resolved_at = $5
		WHERE id = $1 AND status = 'open'`, id, status, resolutionNote, resolvedBy, resolvedAt)
	if err != nil {
		return fmt.Errorf("close rank request: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count closed rank requests: %w", err)
	}
	if n == 0 {
		if _, err := r.Get(ctx, id); err != nil {
			return fmt.Errorf("load rank request that was not closed: %w", err)
		}
		return ErrNotOpen
	}
	return nil
}

// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
func inUTC(req Request) Request {
	req.CreatedAt = req.CreatedAt.UTC()
	if req.ResolvedAt != nil {
		utc := req.ResolvedAt.UTC()
		req.ResolvedAt = &utc
	}
	return req
}
