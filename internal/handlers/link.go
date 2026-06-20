package handlers

import (
	"encoding/json"
	"net/http"
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

func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")

	originalURL, err := h.service.GetByShortCode(shortCode)
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