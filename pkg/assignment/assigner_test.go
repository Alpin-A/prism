package assignment

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fiftyFifty() []Variant {
	return []Variant{
		{ID: "control", Weight: 0.5},
		{ID: "treatment", Weight: 0.5},
	}
}

// assertDistribution assigns n unique users and checks that each variant's
// observed frequency is within tolerancePct of its configured weight.
func assertDistribution(t *testing.T, experimentID string, variants []Variant, n int, tolerancePct float64) {
	t.Helper()
	counts := make(map[string]int, len(variants))
	for i := 0; i < n; i++ {
		v, err := Assign(experimentID, fmt.Sprintf("user_%d", i), variants)
		require.NoError(t, err)
		counts[v]++
	}
	for _, variant := range variants {
		actual := float64(counts[variant.ID]) / float64(n)
		assert.InDelta(t, variant.Weight, actual, tolerancePct,
			"variant %q: expected %.4f, got %.4f", variant.ID, variant.Weight, actual)
	}
}

func TestAssignDeterminism(t *testing.T) {
	variants := fiftyFifty()
	for i := 0; i < 500; i++ {
		userID := fmt.Sprintf("user_%d", i)
		first, err := Assign("exp_homepage", userID, variants)
		require.NoError(t, err)

		for j := 0; j < 10; j++ {
			got, err := Assign("exp_homepage", userID, variants)
			require.NoError(t, err)
			assert.Equal(t, first, got,
				"user %s got %q then %q on repeat call %d", userID, first, got, j)
		}
	}
}

// Passing variants in a different order must produce the same assignment.
// This is enforced by sorting variants by ID inside Assign.
func TestAssignDeterminismVariantOrdering(t *testing.T) {
	variantsAB := []Variant{{ID: "control", Weight: 0.5}, {ID: "treatment", Weight: 0.5}}
	variantsBA := []Variant{{ID: "treatment", Weight: 0.5}, {ID: "control", Weight: 0.5}}

	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user_%d", i)
		got1, err := Assign("exp_order", userID, variantsAB)
		require.NoError(t, err)
		got2, err := Assign("exp_order", userID, variantsBA)
		require.NoError(t, err)
		assert.Equal(t, got1, got2,
			"user %s: variant ordering changed assignment (%q vs %q)", userID, got1, got2)
	}
}

func TestAssignDistribution5050(t *testing.T) {
	assertDistribution(t, "exp_5050", fiftyFifty(), 100_000, 0.01)
}

func TestAssignDistribution9010(t *testing.T) {
	assertDistribution(t, "exp_9010", []Variant{
		{ID: "control", Weight: 0.9},
		{ID: "treatment", Weight: 0.1},
	}, 100_000, 0.01)
}

func TestAssignDistribution199(t *testing.T) {
	// 1% split needs a larger N to land within 0.5% of the target.
	assertDistribution(t, "exp_199", []Variant{
		{ID: "control", Weight: 0.99},
		{ID: "treatment", Weight: 0.01},
	}, 300_000, 0.005)
}

func TestAssignDistributionThreeArm(t *testing.T) {
	assertDistribution(t, "exp_3arm", []Variant{
		{ID: "control", Weight: 0.334},
		{ID: "treatment_a", Weight: 0.333},
		{ID: "treatment_b", Weight: 0.333},
	}, 300_000, 0.01)
}

// For a 50/50 split, the probability that the same user lands in the same
// bucket across two independent experiments is ~50%. A significantly different
// overlap rate would indicate the hashes are correlated.
func TestAssignExperimentIsolation(t *testing.T) {
	variants := fiftyFifty()
	same := 0
	n := 10_000

	for i := 0; i < n; i++ {
		userID := fmt.Sprintf("user_%d", i)
		v1, err := Assign("exp_search_ranking", userID, variants)
		require.NoError(t, err)
		v2, err := Assign("exp_checkout_flow", userID, variants)
		require.NoError(t, err)
		if v1 == v2 {
			same++
		}
	}

	overlap := float64(same) / float64(n)
	assert.InDelta(t, 0.5, overlap, 0.03,
		"assignments correlated across experiments: overlap=%.3f (expected ~0.500)", overlap)
}

func TestAssignUserGetsIndependentVariantsAcrossExperiments(t *testing.T) {
	variants := fiftyFifty()
	counts := make(map[string]int)
	for i := 0; i < 100; i++ {
		v, err := Assign(fmt.Sprintf("exp_%d", i), "power_user_001", variants)
		require.NoError(t, err)
		counts[v]++
	}
	assert.Greater(t, counts["control"], 0, "user never assigned to control across 100 experiments")
	assert.Greater(t, counts["treatment"], 0, "user never assigned to treatment across 100 experiments")
}

func TestValidateNoVariants(t *testing.T) {
	_, err := Assign("exp_1", "user_1", []Variant{})
	assert.Error(t, err)
}

func TestValidateEmptyVariantID(t *testing.T) {
	_, err := Assign("exp_1", "user_1", []Variant{
		{ID: "", Weight: 0.5},
		{ID: "treatment", Weight: 0.5},
	})
	assert.Error(t, err)
}

func TestValidateZeroWeight(t *testing.T) {
	_, err := Assign("exp_1", "user_1", []Variant{
		{ID: "control", Weight: 1.0},
		{ID: "treatment", Weight: 0.0},
	})
	assert.Error(t, err)
}

func TestValidateNegativeWeight(t *testing.T) {
	_, err := Assign("exp_1", "user_1", []Variant{
		{ID: "control", Weight: 1.5},
		{ID: "treatment", Weight: -0.5},
	})
	assert.Error(t, err)
}

func TestValidateWeightsSumTooLow(t *testing.T) {
	_, err := Assign("exp_1", "user_1", []Variant{
		{ID: "control", Weight: 0.4},
		{ID: "treatment", Weight: 0.4},
	})
	assert.Error(t, err)
}

func TestValidateWeightsSumTooHigh(t *testing.T) {
	_, err := Assign("exp_1", "user_1", []Variant{
		{ID: "control", Weight: 0.6},
		{ID: "treatment", Weight: 0.6},
	})
	assert.Error(t, err)
}

// 1/3 + 1/3 + 1/3 = 0.9999999999999999 in float64, which should be accepted.
func TestValidateFloatingPointTolerance(t *testing.T) {
	_, err := Assign("exp_thirds", "user_1", []Variant{
		{ID: "a", Weight: 1.0 / 3.0},
		{ID: "b", Weight: 1.0 / 3.0},
		{ID: "c", Weight: 1.0 / 3.0},
	})
	assert.NoError(t, err)
}

func TestValidateWeightSumOutsideTolerance(t *testing.T) {
	_, err := Assign("exp_1", "user_1", []Variant{
		{ID: "control", Weight: 0.5},
		{ID: "treatment", Weight: 0.4999},
	})
	assert.Error(t, err)
}

func BenchmarkAssign(b *testing.B) {
	variants := fiftyFifty()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Assign("exp_homepage_cta", fmt.Sprintf("user_%d", i), variants) //nolint:errcheck
	}
}

func BenchmarkAssignSameUser(b *testing.B) {
	variants := fiftyFifty()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Assign("exp_homepage_cta", "user_constant_001", variants) //nolint:errcheck
	}
}

func BenchmarkAssignParallel(b *testing.B) {
	variants := fiftyFifty()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Assign("exp_homepage_cta", fmt.Sprintf("user_%d", i), variants) //nolint:errcheck
			i++
		}
	})
}

func TestAssignOutputSample(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sample output in short mode")
	}
	variants := fiftyFifty()
	counts := make(map[string]int)
	for i := 0; i < 20; i++ {
		userID := fmt.Sprintf("user_%04d", i)
		v, _ := Assign("exp_sample", userID, variants)
		counts[v]++
		t.Logf("%-12s → %s", userID, v)
	}
	t.Logf("counts: %v", counts)
	assert.Greater(t, counts["control"], 0)
	assert.Greater(t, counts["treatment"], 0)
}

func TestAssignDeltaFromExpectedWeight(t *testing.T) {
	variants := []Variant{
		{ID: "control", Weight: 0.9},
		{ID: "treatment", Weight: 0.1},
	}
	counts := make(map[string]int)
	n := 1_000_000
	for i := 0; i < n; i++ {
		v, _ := Assign("exp_large_n", fmt.Sprintf("user_%d", i), variants)
		counts[v]++
	}
	for _, variant := range variants {
		actual := float64(counts[variant.ID]) / float64(n)
		delta := math.Abs(actual - variant.Weight)
		t.Logf("variant %q: expected %.4f, got %.4f, delta=%.6f", variant.ID, variant.Weight, actual, delta)
		assert.Less(t, delta, 0.005)
	}
}
