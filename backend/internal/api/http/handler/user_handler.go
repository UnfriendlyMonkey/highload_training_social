package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/api/http/httputil"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, nil)
		return
	}

	userID, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidValue) {
			httputil.WriteJSON(w, http.StatusBadRequest, nil)
			return
		}
		httputil.WriteJSON(w, http.StatusInternalServerError, nil)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.RegisterResponse{UserID: userID})
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.svc.Get(r.Context(), id) // TODO: convert here to dto.UserResponse ??
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

	httputil.WriteJSON(w, http.StatusOK, user)
}
