package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/Alpin-A/prism/pkg/assignment"
	"github.com/Alpin-A/prism/pkg/experiment"
	"github.com/Alpin-A/prism/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	store     *experiment.Store
	publisher *metrics.Publisher
}

func NewHandler(store *experiment.Store, publisher *metrics.Publisher) *Handler {
	return &Handler{store: store, publisher: publisher}
}

func (h *Handler) createExperiment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID          string                `json:"id"`
		Name        string                `json:"name"`
		Description string                `json:"description"`
		MetricType  experiment.MetricType `json:"metric_type"`
		Variants    []struct {
			ID     string  `json:"id"`
			Name   string  `json:"name"`
			Weight float64 `json:"weight"`
		} `json:"variants"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.ID == "" || body.Name == "" || len(body.Variants) == 0 {
		writeError(w, http.StatusBadRequest, "id, name, and variants are required")
		return
	}

	switch body.MetricType {
	case experiment.MetricTypeConversion, experiment.MetricTypeRevenue, experiment.MetricTypeCount:
	default:
		writeError(w, http.StatusBadRequest, "metric_type must be one of: conversion, revenue, count")
		return
	}

	var weightSum float64
	for _, v := range body.Variants {
		weightSum += v.Weight
	}
	if math.Abs(weightSum-1.0) > 1e-9 {
		writeError(w, http.StatusBadRequest, "variant weights must sum to 1.0")
		return
	}

	now := time.Now().UTC()
	exp := experiment.ExperimentWithVariants{
		Experiment: experiment.Experiment{
			ID:          body.ID,
			Name:        body.Name,
			Description: body.Description,
			Status:      experiment.StatusActive,
			MetricType:  body.MetricType,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, v := range body.Variants {
		exp.Variants = append(exp.Variants, experiment.Variant{
			ExperimentID: body.ID,
			ID:           v.ID,
			Name:         v.Name,
			Weight:       v.Weight,
		})
	}

	if err := h.store.Create(r.Context(), exp); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create experiment")
		return
	}

	writeJSON(w, http.StatusCreated, exp)
}

func (h *Handler) getExperiment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	exp, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "experiment not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get experiment")
		}
		return
	}

	writeJSON(w, http.StatusOK, exp)
}

func (h *Handler) listExperiments(w http.ResponseWriter, r *http.Request) {
	experiments, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list experiments")
		return
	}

	writeJSON(w, http.StatusOK, experiments)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		Status experiment.Status `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	switch body.Status {
	case experiment.StatusDraft, experiment.StatusActive, experiment.StatusPaused, experiment.StatusConcluded:
	default:
		writeError(w, http.StatusBadRequest, "status must be one of: draft, active, paused, concluded")
		return
	}

	if err := h.store.UpdateStatus(r.Context(), id, body.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "experiment not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to update status")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	experimentID := r.URL.Query().Get("experiment_id")
	userID := r.URL.Query().Get("user_id")

	if experimentID == "" || userID == "" {
		writeError(w, http.StatusBadRequest, "experiment_id and user_id are required")
		return
	}

	exp, err := h.store.Get(r.Context(), experimentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "experiment not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get experiment")
		}
		return
	}

	variants := make([]assignment.Variant, len(exp.Variants))
	for i, v := range exp.Variants {
		variants[i] = assignment.Variant{ID: v.ID, Weight: v.Weight}
	}

	variantID, err := assignment.Assign(experimentID, userID, variants)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute assignment")
		return
	}

	assignmentsTotal.WithLabelValues(experimentID, variantID).Inc()

	writeJSON(w, http.StatusOK, map[string]string{
		"experiment_id": experimentID,
		"user_id":       userID,
		"variant_id":    variantID,
	})
}

// publishEvent handles POST /api/v1/events.
func (h *Handler) publishEvent(w http.ResponseWriter, r *http.Request) {
	var event metrics.MetricEvent

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if event.ExperimentID == "" || event.UserID == "" || event.VariantID == "" || event.EventType == "" {
		writeError(w, http.StatusBadRequest, "experiment_id, user_id, variant_id, and event_type are required")
		return
	}

	if event.Value == 0 {
		event.Value = 1.0
	}

	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	if err := h.publisher.Publish(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to publish event")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
