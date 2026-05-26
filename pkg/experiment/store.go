package experiment

import (
	"context"
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

// Create inserts a new experiment and its variants in a single transaction.
func (s *Store) Create(ctx context.Context, e ExperimentWithVariants) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO experiments (id, name, description, status, metric_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, e.ID, e.Name, e.Description, e.Status, e.MetricType, e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting experiment: %w", err)
	}

	for _, v := range e.Variants {
		_, err = tx.Exec(ctx, `
			INSERT INTO variants (experiment_id, id, name, weight)
			VALUES ($1, $2, $3, $4)
		`, e.ID, v.ID, v.Name, v.Weight)
		if err != nil {
			return fmt.Errorf("inserting variant %q: %w", v.ID, err)
		}
	}

	return tx.Commit(ctx)
}

// Get returns an error wrapping pgx.ErrNoRows if the experiment does not exist.
func (s *Store) Get(ctx context.Context, id string) (ExperimentWithVariants, error) {
	var e ExperimentWithVariants

	err := s.db.QueryRow(ctx, `
		SELECT id, name, description, status, metric_type, created_at, updated_at
		FROM experiments
		WHERE id = $1
	`, id).Scan(
		&e.ID, &e.Name, &e.Description, &e.Status, &e.MetricType, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return e, fmt.Errorf("querying experiment %q: %w", id, err)
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, name, weight
		FROM variants
		WHERE experiment_id = $1
		ORDER BY id
	`, id)
	if err != nil {
		return e, fmt.Errorf("querying variants for %q: %w", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.ID, &v.Name, &v.Weight); err != nil {
			return e, fmt.Errorf("scanning variant: %w", err)
		}
		v.ExperimentID = id
		e.Variants = append(e.Variants, v)
	}

	return e, rows.Err()
}

func (s *Store) UpdateStatus(ctx context.Context, id string, status Status) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE experiments
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("updating status for %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("updating status for %q: %w", id, pgx.ErrNoRows)
	}
	return nil
}

func (s *Store) RecordExposure(ctx context.Context, experimentID, userID, variantID string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO exposures (experiment_id, user_id, variant_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (experiment_id, user_id) DO NOTHING
	`, experimentID, userID, variantID)
	if err != nil {
		return fmt.Errorf("recording exposure: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]ExperimentWithVariants, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, description, status, metric_type, created_at, updated_at
		FROM experiments
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing experiments: %w", err)
	}
	defer rows.Close()

	var experiments []ExperimentWithVariants
	for rows.Next() {
		var e ExperimentWithVariants
		if err := rows.Scan(
			&e.ID, &e.Name, &e.Description, &e.Status, &e.MetricType, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning experiment: %w", err)
		}
		experiments = append(experiments, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// N+1: one extra query per experiment to load variants. Fine at low experiment
	// counts; replace with a single JOIN query if this becomes a bottleneck.
	for i, e := range experiments {
		full, err := s.Get(ctx, e.ID)
		if err != nil {
			return nil, err
		}
		experiments[i].Variants = full.Variants
	}

	return experiments, nil
}
