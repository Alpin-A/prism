package experiment

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusConcluded Status = "concluded"
)

type MetricType string

const (
	MetricTypeConversion MetricType = "conversion"
	MetricTypeRevenue    MetricType = "revenue"
	MetricTypeCount      MetricType = "count"
)

type Experiment struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	MetricType  MetricType `json:"metric_type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Variant struct {
	ExperimentID string  `json:"experiment_id"`
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
}

// ExperimentWithVariants is the typical unit for creating or reading an experiment.
type ExperimentWithVariants struct {
	Experiment
	Variants []Variant `json:"variants"`
}
