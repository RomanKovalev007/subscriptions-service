package transport

import (
	"encoding/json"
	"net/http"

	"github.com/RomanKovalev007/subscriptions-service/internal/apperr"
	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

// createSubscription godoc
// @Summary      Create subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        input body domain.CreateSubscriptionInput true "Subscription data"
// @Success      201 {object} domain.Subscription
// @Failure      400 {object} domain.ErrorResponse
// @Failure      500 {object} domain.ErrorResponse
// @Router       /subscriptions [post]
func (h *Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "invalid request body")
		return
	}
	if err := validate.Struct(input); err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, err.Error())
		return
	}

	sub, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleAppErr(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, sub)
}

// listSubscriptions godoc
// @Summary      List subscriptions
// @Tags         subscriptions
// @Produce      json
// @Param        user_id      query string true  "User UUID"
// @Param        service_name query string false "Filter by service name"
// @Success      200 {array}  domain.Subscription
// @Failure      400 {object} domain.ErrorResponse
// @Failure      500 {object} domain.ErrorResponse
// @Router       /subscriptions [get]
func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.URL.Query().Get("user_id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "user_id is required and must be a valid UUID")
		return
	}

	var serviceName *string
	if sn := r.URL.Query().Get("service_name"); sn != "" {
		serviceName = &sn
	}

	subs, err := h.svc.List(r.Context(), userID, serviceName)
	if err != nil {
		h.handleAppErr(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, subs)
}

// getSubscription godoc
// @Summary      Get subscription by ID
// @Tags         subscriptions
// @Produce      json
// @Param        id path string true "Subscription UUID"
// @Success      200 {object} domain.Subscription
// @Failure      400 {object} domain.ErrorResponse
// @Failure      404 {object} domain.ErrorResponse
// @Failure      500 {object} domain.ErrorResponse
// @Router       /subscriptions/{id} [get]
func (h *Handler) getSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "invalid id")
		return
	}

	sub, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleAppErr(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, sub)
}

// updateSubscription godoc
// @Summary      Update subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id    path string                        true "Subscription UUID"
// @Param        input body domain.UpdateSubscriptionInput true "Subscription data"
// @Success      200 {object} domain.Subscription
// @Failure      400 {object} domain.ErrorResponse
// @Failure      404 {object} domain.ErrorResponse
// @Failure      500 {object} domain.ErrorResponse
// @Router       /subscriptions/{id} [put]
func (h *Handler) updateSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "invalid id")
		return
	}

	var input domain.UpdateSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "invalid request body")
		return
	}
	if err := validate.Struct(input); err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, err.Error())
		return
	}

	sub, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleAppErr(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, sub)
}

// deleteSubscription godoc
// @Summary      Delete subscription
// @Tags         subscriptions
// @Param        id path string true "Subscription UUID"
// @Success      204
// @Failure      400 {object} domain.ErrorResponse
// @Failure      404 {object} domain.ErrorResponse
// @Failure      500 {object} domain.ErrorResponse
// @Router       /subscriptions/{id} [delete]
func (h *Handler) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleAppErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// totalCost godoc
// @Summary      Calculate total subscription cost
// @Tags         subscriptions
// @Produce      json
// @Param        from         query string true  "Start month (MM-YYYY)"
// @Param        to           query string true  "End month (MM-YYYY)"
// @Param        user_id      query string true  "User UUID"
// @Param        service_name query string false "Filter by service name"
// @Success      200 {object} map[string]int
// @Failure      400 {object} domain.ErrorResponse
// @Failure      500 {object} domain.ErrorResponse
// @Router       /subscriptions/total-cost [get]
func (h *Handler) totalCost(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "from and to query params are required (MM-YYYY)")
		return
	}

	raw := r.URL.Query().Get("user_id")
	if raw == "" {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "user_id param is required")
		return
	}
	userID, err := uuid.Parse(raw)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "invalid user_id")
		return
	}

	var serviceName *string
	if sn := r.URL.Query().Get("service_name"); sn != "" {
		serviceName = &sn
	}

	total, err := h.svc.TotalCost(r.Context(), from, to, userID, serviceName)
	if err != nil {
		h.handleAppErr(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]int{"total_cost": total})
}
