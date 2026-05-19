package api

import (
	"github.com/Alpin-A/prism/pkg/experiment"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(store *experiment.Store) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(metricsMiddleware)

	// Prometheus scrapes this endpoint every 15 seconds.
	r.Handle("/metrics", promhttp.Handler())

	h := NewHandler(store)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/experiments", h.createExperiment)
		r.Get("/experiments", h.listExperiments)
		r.Get("/experiments/{id}", h.getExperiment)
		r.Patch("/experiments/{id}/status", h.updateStatus)
		r.Get("/assign", h.assign)
	})

	return r
}
