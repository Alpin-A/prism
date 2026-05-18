package experiment

import "time"

// Status represents the lifecycle stage of an experiment.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusConcluded Status = "concluded"
)

// MetricType defines what kind of metric an experiment measures.
type MetricType string

const (
	MetricTypeConversion MetricType = "conversion"
	MetricTypeRevenue    MetricType = "revenue"
	MetricTypeCount      MetricType = "count"
)

// Experiment represents a single A/B test.
type Experiment struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	MetricType  MetricType `json:"metric_type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Variant represents a single arm of an experiment.
type Variant struct {
	ExperimentID string  `json:"experiment_id"`
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
}

// ExperimentWithVariants groups an experiment with its variants,
// which is the typical unit you work with when creating or reading an experiment.
type ExperimentWithVariants struct {
	Experiment
	Variants []Variant `json:"variants"`
}
