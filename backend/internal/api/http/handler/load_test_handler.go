package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/api/http/httputil"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
)

type LoadTestHandler struct {
	svc *service.LoadTestService
}

func NewLoadTestHandler(svc *service.LoadTestService) *LoadTestHandler {
	return &LoadTestHandler{svc: svc}
}

func (h *LoadTestHandler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	var req dto.LoadTestEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, nil)
		return
	}

	id, err := h.svc.RecordEvent(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidValue) {
			httputil.WriteJSON(w, http.StatusBadRequest, nil)
			return
		}
		httputil.WriteJSON(w, http.StatusInternalServerError, nil)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, dto.LoadTestEventResponse{ID: id})
}

func (h *LoadTestHandler) Count(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")
	count, err := h.svc.Count(r.Context(), runID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidValue) {
			httputil.WriteJSON(w, http.StatusBadRequest, nil)
			return
		}
		httputil.WriteJSON(w, http.StatusInternalServerError, nil)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.LoadTestCountResponse{
		RunID: runID,
		Count: count,
	})
}
