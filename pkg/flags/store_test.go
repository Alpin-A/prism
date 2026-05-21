package flags

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
		CREATE TABLE feature_flags (
			id          VARCHAR(128) PRIMARY KEY,
			name        TEXT NOT NULL,
			enabled     BOOLEAN NOT NULL DEFAULT FALSE,
			rollout_pct NUMERIC(5,2) NOT NULL DEFAULT 0.0
			            CHECK (rollout_pct BETWEEN 0 AND 100),
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE flag_overrides (
			flag_id  VARCHAR(128) NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
			user_id  VARCHAR(255) NOT NULL,
			enabled  BOOLEAN NOT NULL,
			PRIMARY KEY (flag_id, user_id)
		)
	`)
	require.NoError(t, err)
}

func makeFlag() Flag {
	return Flag{
		ID:         "dark_mode",
		Name:       "Dark Mode",
		Enabled:    true,
		RolloutPct: 50.0,
		CreatedAt:  time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:  time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestFlagCreate(t *testing.T) {
	store := NewStore(setupTestDB(t))
	err := store.Create(context.Background(), makeFlag())
	require.NoError(t, err)
}

func TestFlagGet(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	flag := makeFlag()
	require.NoError(t, store.Create(ctx, flag))

	got, err := store.Get(ctx, flag.ID)
	require.NoError(t, err)

	assert.Equal(t, flag.ID, got.ID)
	assert.Equal(t, flag.Name, got.Name)
	assert.Equal(t, flag.Enabled, got.Enabled)
	assert.Equal(t, flag.RolloutPct, got.RolloutPct)
}

func TestFlagGetNotFound(t *testing.T) {
	store := NewStore(setupTestDB(t))
	_, err := store.Get(context.Background(), "does_not_exist")
	assert.Error(t, err)
}

func TestFlagUpdate(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, makeFlag()))
	require.NoError(t, store.Update(ctx, "dark_mode", false, 25.0))

	got, err := store.Get(ctx, "dark_mode")
	require.NoError(t, err)
	assert.Equal(t, false, got.Enabled)
	assert.Equal(t, 25.0, got.RolloutPct)
}

func TestFlagList(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	flag1 := makeFlag()
	flag2 := makeFlag()
	flag2.ID = "new_checkout"
	flag2.Name = "New Checkout Flow"

	require.NoError(t, store.Create(ctx, flag1))
	require.NoError(t, store.Create(ctx, flag2))

	flags, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, flags, 2)
}

func TestFlagCreateDuplicate(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	flag := makeFlag()
	require.NoError(t, store.Create(ctx, flag))

	err := store.Create(ctx, flag)
	assert.Error(t, err)
}

func TestOverrideSetAndGet(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, makeFlag()))

	override := Override{FlagID: "dark_mode", UserID: "user_001", Enabled: true}
	require.NoError(t, store.SetOverride(ctx, override))

	got, found, err := store.GetOverride(ctx, "dark_mode", "user_001")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, true, got.Enabled)
}

func TestOverrideNotFound(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, makeFlag()))

	_, found, err := store.GetOverride(ctx, "dark_mode", "user_nobody")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestOverrideUpdate(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, makeFlag()))

	// Set override to true then update to false
	require.NoError(t, store.SetOverride(ctx, Override{FlagID: "dark_mode", UserID: "user_001", Enabled: true}))
	require.NoError(t, store.SetOverride(ctx, Override{FlagID: "dark_mode", UserID: "user_001", Enabled: false}))

	got, found, err := store.GetOverride(ctx, "dark_mode", "user_001")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, false, got.Enabled)
}
