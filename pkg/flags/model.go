package flags

import "time"

// Flag is a feature flag with percentage-based rollout.
type Flag struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	RolloutPct float64   `json:"rollout_pct"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Override forces a specific user to see a flag as enabled or disabled, bypassing the rollout percentage.
type Override struct {
	FlagID  string `json:"flag_id"`
	UserID  string `json:"user_id"`
	Enabled bool   `json:"enabled"`
}

// EvalResult is the result of evaluating a flag for a specific user.
type EvalResult struct {
	FlagID  string `json:"flag_id"`
	UserID  string `json:"user_id"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}
