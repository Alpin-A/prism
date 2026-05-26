package api

import (
	"github.com/Alpin-A/prism/pkg/experiment"
	flagspkg "github.com/Alpin-A/prism/pkg/flags"
	"github.com/Alpin-A/prism/pkg/metrics"
	"github.com/Alpin-A/prism/pkg/statsclient"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(
	store *experiment.Store,
	publisher *metrics.Publisher,
	statsClient *statsclient.Client,
	flagStore *flagspkg.Store,
	flagEvaluator *flagspkg.Evaluator,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(metricsMiddleware)

	r.Handle("/metrics", promhttp.Handler())

	h := NewHandler(store, publisher, statsClient, flagStore, flagEvaluator)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/experiments", h.createExperiment)
		r.Get("/experiments", h.listExperiments)
		r.Get("/experiments/{id}", h.getExperiment)
		r.Patch("/experiments/{id}/status", h.updateStatus)
		r.Get("/experiments/{id}/results", h.getExperimentResults)
		r.Get("/assign", h.assign)
		r.Post("/events", h.publishEvent)
		r.Post("/flags", h.createFlag)
		r.Get("/flags", h.listFlags)
		r.Get("/flags/{id}/evaluate", h.evaluateFlag)
		r.Patch("/flags/{id}", h.updateFlag)
	})

	return r
}
