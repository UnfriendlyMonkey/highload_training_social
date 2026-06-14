package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/api/http/httputil"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, nil)
		return
	}

	token, err := h.svc.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidValue):
			httputil.WriteJSON(w, http.StatusBadRequest, nil)
		case errors.Is(err, service.ErrNotFound):
			httputil.WriteJSON(w, http.StatusNotFound, nil)
		default:
			httputil.WriteJSON(w, http.StatusInternalServerError, nil)
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.LoginResponse{Token: token})
}
