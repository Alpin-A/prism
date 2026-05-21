package flags

import "time"

// Represents a feature flag with a percentage-based rollout ranging from 0.0 to 100.0.
type Flag struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	RolloutPct float64   `json:"rollout_pct"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Forces a specific user to see a flag as on or off, regardless of the rollout percentage
type Override struct {
	FlagID  string `json:"flag_id"`
	UserID  string `json:"user_id"`
	Enabled bool   `json:"enabled"`
}

// Result of evaluating a flag for a user (override, rollout, or disabled)
type EvalResult struct {
	FlagID  string `json:"flag_id"`
	UserID  string `json:"user_id"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}
