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

func (h *Handler) totalCost(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "from and to query params are required (MM-YYYY)")
		return
	}

	var userID uuid.UUID
	raw := r.URL.Query().Get("user_id")
	if raw == "" {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "user_id param is required")
		return
	} 
	id, err := uuid.Parse(raw)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apperr.CodeInvalidInput, "invalid user_id")
		return
	}
	userID = id
	

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