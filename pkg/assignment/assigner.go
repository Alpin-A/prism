package assignment

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// weightTolerance accounts for floating-point rounding when weights are
// computed as fractions, e.g. 1/3 + 1/3 + 1/3 = 0.9999999999999999.
const weightTolerance = 1e-9

type Variant struct {
	ID     string
	Weight float64 // must be > 0; all weights in an experiment must sum to 1.0
}

// Assign returns the variant a user belongs to for a given experiment.
//
// The assignment is computed by hashing "experimentID:userID" with SHA-256
// and mapping the result onto the variants' cumulative weight buckets.
// Including the experimentID in the hash means the same user gets an
// independent assignment for each experiment they are enrolled in.
//
// Variants are sorted by ID before bucketing so that the result does not
// depend on the order the caller passes them in.
func Assign(experimentID, userID string, variants []Variant) (string, error) {
	if err := validate(variants); err != nil {
		return "", err
	}

	sorted := make([]Variant, len(variants))
	copy(sorted, variants)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", experimentID, userID)))

	u := binary.BigEndian.Uint64(digest[:8])
	normalized := float64(u) / float64(math.MaxUint64)

	cumulative := 0.0
	for _, v := range sorted {
		cumulative += v.Weight
		if normalized < cumulative {
			return v.ID, nil
		}
	}

	// Floating-point rounding can prevent cumulative from reaching exactly 1.0,
	// so normalized may not fall into any bucket. Return the last variant.
	return sorted[len(sorted)-1].ID, nil
}

func validate(variants []Variant) error {
	if len(variants) == 0 {
		return fmt.Errorf("assignment: at least one variant is required")
	}

	sum := 0.0
	for _, v := range variants {
		if v.ID == "" {
			return fmt.Errorf("assignment: variant ID must not be empty")
		}
		if v.Weight <= 0 {
			return fmt.Errorf("assignment: variant %q has non-positive weight %.6f", v.ID, v.Weight)
		}
		sum += v.Weight
	}

	if math.Abs(sum-1.0) > weightTolerance {
		return fmt.Errorf("assignment: weights sum to %.10f, must be 1.0 (±%.0e)", sum, weightTolerance)
	}

	return nil
}
