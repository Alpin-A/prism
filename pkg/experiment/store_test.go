package experiment

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB spins up a real Postgres instance in Docker for the duration
// of the test, runs migrations, and returns a connection pool.
// The container is cleaned up automatically when the test finishes.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("prism_test"),
		postgres.WithUsername("prism"),
		postgres.WithPassword("prism_test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		container.Terminate(ctx)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	runMigrations(t, ctx, pool)

	return pool
}

func runMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		CREATE TABLE experiments (
			id          VARCHAR(128) PRIMARY KEY,
			name        TEXT NOT NULL,
			description TEXT,
			status      VARCHAR(32) NOT NULL DEFAULT 'draft'
			            CHECK (status IN ('draft', 'active', 'paused', 'concluded')),
			metric_type VARCHAR(64) NOT NULL DEFAULT 'conversion',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE variants (
			experiment_id  VARCHAR(128) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
			id             VARCHAR(128) NOT NULL,
			name           TEXT NOT NULL,
			weight         NUMERIC(6,5) NOT NULL CHECK (weight > 0 AND weight <= 1),
			PRIMARY KEY (experiment_id, id)
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE exposures (
			experiment_id  VARCHAR(128) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
			user_id        VARCHAR(255) NOT NULL,
			variant_id     VARCHAR(128) NOT NULL,
			first_seen     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (experiment_id, user_id)
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE assignment_overrides (
			experiment_id  VARCHAR(128) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
			user_id        VARCHAR(255) NOT NULL,
			variant_id     VARCHAR(128) NOT NULL,
			PRIMARY KEY (experiment_id, user_id)
		)
	`)
	require.NoError(t, err)
}

func makeExperiment() ExperimentWithVariants {
	return ExperimentWithVariants{
		Experiment: Experiment{
			ID:          "exp_homepage_cta",
			Name:        "Homepage CTA Button Color",
			Description: "Testing blue vs green button",
			Status:      StatusActive,
			MetricType:  MetricTypeConversion,
			CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
			UpdatedAt:   time.Now().UTC().Truncate(time.Millisecond),
		},
		Variants: []Variant{
			{ExperimentID: "exp_homepage_cta", ID: "control", Name: "Blue Button", Weight: 0.5},
			{ExperimentID: "exp_homepage_cta", ID: "treatment", Name: "Green Button", Weight: 0.5},
		},
	}
}

func TestStoreCreate(t *testing.T) {
	pool := setupTestDB(t)
	store := NewStore(pool)
	ctx := context.Background()

	exp := makeExperiment()
	err := store.Create(ctx, exp)
	require.NoError(t, err)
}

func TestStoreGet(t *testing.T) {
	pool := setupTestDB(t)
	store := NewStore(pool)
	ctx := context.Background()

	exp := makeExperiment()
	require.NoError(t, store.Create(ctx, exp))

	got, err := store.Get(ctx, exp.ID)
	require.NoError(t, err)

	assert.Equal(t, exp.ID, got.ID)
	assert.Equal(t, exp.Name, got.Name)
	assert.Equal(t, exp.Status, got.Status)
	assert.Equal(t, exp.MetricType, got.MetricType)
	assert.Len(t, got.Variants, 2)
}

func TestStoreGetNotFound(t *testing.T) {
	pool := setupTestDB(t)
	store := NewStore(pool)
	ctx := context.Background()

	_, err := store.Get(ctx, "does_not_exist")
	assert.Error(t, err)
}

func TestStoreUpdateStatus(t *testing.T) {
	pool := setupTestDB(t)
	store := NewStore(pool)
	ctx := context.Background()

	exp := makeExperiment()
	require.NoError(t, store.Create(ctx, exp))
	require.NoError(t, store.UpdateStatus(ctx, exp.ID, StatusPaused))

	got, err := store.Get(ctx, exp.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, got.Status)
}

func TestStoreList(t *testing.T) {
	pool := setupTestDB(t)
	store := NewStore(pool)
	ctx := context.Background()

	exp1 := makeExperiment()
	exp2 := makeExperiment()
	exp2.ID = "exp_checkout_flow"
	exp2.Name = "Checkout Flow Test"

	require.NoError(t, store.Create(ctx, exp1))
	require.NoError(t, store.Create(ctx, exp2))

	experiments, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, experiments, 2)
}

func TestStoreCreateDuplicateID(t *testing.T) {
	pool := setupTestDB(t)
	store := NewStore(pool)
	ctx := context.Background()

	exp := makeExperiment()
	require.NoError(t, store.Create(ctx, exp))

	err := store.Create(ctx, exp)
	assert.Error(t, err)
}
