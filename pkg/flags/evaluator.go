package flags

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
)

type Evaluator struct {
	store *Store
}

func NewEvaluator(store *Store) *Evaluator {
	return &Evaluator{store: store}
}

func (e *Evaluator) Evaluate(ctx context.Context, flagID, userID string) (EvalResult, error) {
	flag, err := e.store.Get(ctx, flagID)
	if err != nil {
		return EvalResult{}, fmt.Errorf("getting flag: %w", err)
	}

	if !flag.Enabled {
		return EvalResult{FlagID: flagID, UserID: userID, Enabled: false, Reason: "disabled"}, nil
	}

	override, found, err := e.store.GetOverride(ctx, flagID, userID)
	if err != nil {
		return EvalResult{}, fmt.Errorf("getting override: %w", err)
	}
	if found {
		return EvalResult{FlagID: flagID, UserID: userID, Enabled: override.Enabled, Reason: "override"}, nil
	}

	if flag.RolloutPct >= 100.0 {
		return EvalResult{FlagID: flagID, UserID: userID, Enabled: true, Reason: "rollout"}, nil
	}

	enabled := hashRollout(flagID, userID, flag.RolloutPct)
	return EvalResult{FlagID: flagID, UserID: userID, Enabled: enabled, Reason: "rollout"}, nil
}

// hashRollout uses the same SHA-256 bucketing approach as the assignment package.
func hashRollout(flagID, userID string, rolloutPct float64) bool {
	key := fmt.Sprintf("flag:%s:%s", flagID, userID)
	digest := sha256.Sum256([]byte(key))
	u := binary.BigEndian.Uint64(digest[:8])
	normalized := float64(u) / float64(math.MaxUint64)
	return normalized < (rolloutPct / 100.0)
}
