package transport

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/RomanKovalev007/subscriptions-service/internal/apperr"
	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
)

var codeToStatus = map[string]int{
	apperr.CodeInternalError: http.StatusInternalServerError,
	apperr.CodeInvalidInput: http.StatusBadRequest,
	apperr.CodeNotFound: http.StatusNotFound,
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, msg string) {
	h.writeJSON(w, status, domain.ErrorResponse{
		Code: code,
		Message: msg,
	})
}

func (h *Handler) handleAppErr(w http.ResponseWriter, err error) {
	var svcErr *apperr.Error
	if errors.As(err, &svcErr){
		status, ok := codeToStatus[svcErr.Code]
		if !ok{
			status = http.StatusInternalServerError
		}
		h.writeError(w, status, svcErr.Code, svcErr.Message)
		return
	}
	h.writeError(w, http.StatusInternalServerError, apperr.CodeInternalError, "internal server error")
}