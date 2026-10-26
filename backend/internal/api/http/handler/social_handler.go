package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/api/http/httputil"
	"github.com/UnfriendlyMonkey/hsn/internal/api/http/middleware"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
	"github.com/go-chi/chi/v5"
)

type SocialHandler struct{ svc *service.SocialService }

func NewSocialHandler(svc *service.SocialService) *SocialHandler { return &SocialHandler{svc: svc} }

func socialError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidValue):
		httputil.WriteJSON(w, http.StatusBadRequest, nil)
	case errors.Is(err, service.ErrNotFound):
		httputil.WriteJSON(w, http.StatusNotFound, nil)
	default:
		httputil.WriteJSON(w, http.StatusInternalServerError, nil)
	}
}
func (h *SocialHandler) SetFriend(w http.ResponseWriter, r *http.Request) { h.changeFriend(w, r, true) }
func (h *SocialHandler) DeleteFriend(w http.ResponseWriter, r *http.Request) {
	h.changeFriend(w, r, false)
}
func (h *SocialHandler) changeFriend(w http.ResponseWriter, r *http.Request, add bool) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, nil)
		return
	}
	var err error
	if add {
		err = h.svc.SetFriend(r.Context(), user, chi.URLParam(r, "user_id"))
	} else {
		err = h.svc.DeleteFriend(r.Context(), user, chi.URLParam(r, "user_id"))
	}
	if err != nil {
		socialError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SocialHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, nil)
		return
	}
	var req dto.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		socialError(w, service.ErrInvalidValue)
		return
	}
	id, err := h.svc.CreatePost(r.Context(), user, req)
	if err != nil {
		socialError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, id)
}
func (h *SocialHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, nil)
		return
	}
	var req dto.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		socialError(w, service.ErrInvalidValue)
		return
	}
	if err := h.svc.UpdatePost(r.Context(), user, req); err != nil {
		socialError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func (h *SocialHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, nil)
		return
	}
	if err := h.svc.DeletePost(r.Context(), user, chi.URLParam(r, "id")); err != nil {
		socialError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func (h *SocialHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	post, err := h.svc.GetPost(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		socialError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, post)
}

func (h *SocialHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, nil)
		return
	}
	offset, limit := 0, 10
	var err error
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			socialError(w, service.ErrInvalidValue)
			return
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			socialError(w, service.ErrInvalidValue)
			return
		}
	}
	posts, err := h.svc.GetFeed(r.Context(), user, offset, limit)
	if err != nil {
		socialError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, posts)
}
