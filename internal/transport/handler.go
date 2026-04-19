package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	svc    SubscriptionService
	logger *slog.Logger
}

func New(svc SubscriptionService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(SlogLogger(h.logger))

	r.Route("/api/v1/subscriptions", func(r chi.Router) {
		r.Post("/", h.createSubscription)
		r.Get("/", h.listSubscriptions)
		r.Get("/total-cost", h.totalCost)
		r.Get("/{id}", h.getSubscription)
		r.Put("/{id}", h.updateSubscription)
		r.Delete("/{id}", h.deleteSubscription)
	})

	return r
}


