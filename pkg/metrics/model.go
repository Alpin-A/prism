package metrics

import "time"

// MetricEvent represents a single user action within an experiment.
// These are published to Kafka by the API and consumed by the metric consumer.
type MetricEvent struct {
	ExperimentID string    `json:"experiment_id"`
	UserID       string    `json:"user_id"`
	VariantID    string    `json:"variant_id"`
	EventType    string    `json:"event_type"` // e.g. "conversion", "click", "purchase"
	Value        float64   `json:"value"`      // 1.0 for binary events, revenue amount for purchases
	OccurredAt   time.Time `json:"occurred_at"`
}
