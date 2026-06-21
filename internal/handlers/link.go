package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"shrt/internal/apierr"
	"shrt/internal/auth"
	"shrt/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type LinkHandler struct {
	service *services.LinkService
}

func NewLinkHandler(service *services.LinkService) *LinkHandler {
	return &LinkHandler{service: service}
}

func (h *LinkHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShortCode   string     `json:"short_code,omitempty"`
		OriginalURL string     `json:"original_url"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.WriteError(w, apierr.NewValidation("invalid request payload"))
		return
	}

	var userID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		userID = &claims.UserID
	}

	url, err := h.service.CreateShortURL(userID, req.ShortCode, req.OriginalURL, req.ExpiresAt)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, url)
}

func (h *LinkHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		apierr.WriteError(w, apierr.NewUnauthorized("missing token"))
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	result, err := h.service.ListLinks(claims.UserID, page, pageSize)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")

	originalURL, err := h.service.GetByShortCode(shortCode, r.Referer())
	if err != nil {
		apierr.WriteError(w, apierr.NewNotFound("short URL not found"))
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func (h *LinkHandler) RetrieveOriginalURL(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		apierr.WriteError(w, apierr.NewUnauthorized("missing token"))
		return
	}

	shortCode := chi.URLParam(r, "shortCode")

	link, err := h.service.RetrieveLink(claims.UserID, shortCode)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, link)
}

func (h *LinkHandler) UpdateShortURL(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		apierr.WriteError(w, apierr.NewUnauthorized("missing token"))
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.WriteError(w, apierr.NewValidation("invalid request payload"))
		return
	}

	shortCode := chi.URLParam(r, "shortCode")

	link, err := h.service.UpdateLink(claims.UserID, shortCode, req.URL)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, link)
}

func (h *LinkHandler) DeleteShortURL(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		apierr.WriteError(w, apierr.NewUnauthorized("missing token"))
		return
	}

	shortCode := chi.URLParam(r, "shortCode")

	if err := h.service.DeleteLink(claims.UserID, shortCode); err != nil {
		apierr.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *LinkHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		apierr.WriteError(w, apierr.NewUnauthorized("missing token"))
		return
	}

	shortCode := chi.URLParam(r, "shortCode")

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats, err := h.service.GetStats(claims.UserID, shortCode, days)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}