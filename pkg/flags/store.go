package flags

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, f Flag) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO feature_flags (id, name, enabled, rollout_pct, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, f.ID, f.Name, f.Enabled, f.RolloutPct, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting flag: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (Flag, error) {
	var f Flag
	err := s.db.QueryRow(ctx, `
		SELECT id, name, enabled, rollout_pct, created_at, updated_at
		FROM feature_flags
		WHERE id = $1
	`, id).Scan(&f.ID, &f.Name, &f.Enabled, &f.RolloutPct, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return f, fmt.Errorf("querying flag %q: %w", id, err)
	}
	return f, nil
}

func (s *Store) Update(ctx context.Context, id string, enabled bool, rolloutPct float64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE feature_flags
		SET enabled = $1, rollout_pct = $2, updated_at = $3
		WHERE id = $4
	`, enabled, rolloutPct, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("updating flag %q: %w", id, err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]Flag, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, enabled, rollout_pct, created_at, updated_at
		FROM feature_flags
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing flags: %w", err)
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.ID, &f.Name, &f.Enabled, &f.RolloutPct, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning flag: %w", err)
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

func (s *Store) GetOverride(ctx context.Context, flagID, userID string) (Override, bool, error) {
	var o Override
	err := s.db.QueryRow(ctx, `
		SELECT flag_id, user_id, enabled
		FROM flag_overrides
		WHERE flag_id = $1 AND user_id = $2
	`, flagID, userID).Scan(&o.FlagID, &o.UserID, &o.Enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return o, false, nil
		}
		return o, false, fmt.Errorf("querying override: %w", err)
	}
	return o, true, nil
}

func (s *Store) SetOverride(ctx context.Context, o Override) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO flag_overrides (flag_id, user_id, enabled)
		VALUES ($1, $2, $3)
		ON CONFLICT (flag_id, user_id) DO UPDATE
		SET enabled = EXCLUDED.enabled
	`, o.FlagID, o.UserID, o.Enabled)
	if err != nil {
		return fmt.Errorf("setting override: %w", err)
	}
	return nil
}
