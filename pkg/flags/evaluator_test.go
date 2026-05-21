package flags

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateDisabledFlag(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	flag := Flag{
		ID: "my_flag", Name: "My Flag",
		Enabled: false, RolloutPct: 100.0,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Create(ctx, flag))

	eval := NewEvaluator(store)
	result, err := eval.Evaluate(ctx, "my_flag", "user_001")
	require.NoError(t, err)

	assert.False(t, result.Enabled)
	assert.Equal(t, "disabled", result.Reason)
}

func TestEvaluateOverrideTrumpsRollout(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	// Flag is enabled but 0% rollout — no user should see it.
	flag := Flag{
		ID: "my_flag", Name: "My Flag",
		Enabled: true, RolloutPct: 0.0,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Create(ctx, flag))

	// Override forces user_001 to see it as enabled.
	require.NoError(t, store.SetOverride(ctx, Override{
		FlagID: "my_flag", UserID: "user_001", Enabled: true,
	}))

	eval := NewEvaluator(store)
	result, err := eval.Evaluate(ctx, "my_flag", "user_001")
	require.NoError(t, err)

	assert.True(t, result.Enabled)
	assert.Equal(t, "override", result.Reason)
}

func TestEvaluateFullRollout(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	flag := Flag{
		ID: "my_flag", Name: "My Flag",
		Enabled: true, RolloutPct: 100.0,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Create(ctx, flag))

	eval := NewEvaluator(store)
	result, err := eval.Evaluate(ctx, "my_flag", "user_001")
	require.NoError(t, err)

	assert.True(t, result.Enabled)
	assert.Equal(t, "rollout", result.Reason)
}

func TestEvaluateRolloutDeterminism(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	flag := Flag{
		ID: "my_flag", Name: "My Flag",
		Enabled: true, RolloutPct: 50.0,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Create(ctx, flag))

	eval := NewEvaluator(store)

	// Same user always gets the same result.
	first, err := eval.Evaluate(ctx, "my_flag", "user_001")
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		result, err := eval.Evaluate(ctx, "my_flag", "user_001")
		require.NoError(t, err)
		assert.Equal(t, first.Enabled, result.Enabled)
	}
}

func TestEvaluateRolloutDistribution(t *testing.T) {
	store := NewStore(setupTestDB(t))
	ctx := context.Background()

	flag := Flag{
		ID: "my_flag", Name: "My Flag",
		Enabled: true, RolloutPct: 50.0,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Create(ctx, flag))

	eval := NewEvaluator(store)
	enabled := 0
	n := 10_000

	for i := 0; i < n; i++ {
		result, err := eval.Evaluate(ctx, "my_flag",
			fmt.Sprintf("user_%d", i))
		require.NoError(t, err)
		if result.Enabled {
			enabled++
		}
	}

	actual := float64(enabled) / float64(n)
	// Should be within 1% of 50%
	assert.InDelta(t, 0.5, actual, 0.01)
}
