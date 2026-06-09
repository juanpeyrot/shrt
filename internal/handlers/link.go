package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"shrt/internal/apierr"
	"shrt/internal/services"
)

type LinkHandler struct {
	service *services.LinkService
}

func NewLinkHandler(service *services.LinkService) *LinkHandler {
	return &LinkHandler{service: service}
}

func (h *LinkHandler) RegisterRoutes(r chi.Router) {
	r.Post("/links", h.CreateShortURL)
	r.Get("/{shortCode}", h.Redirect)
}

func (h *LinkHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShortCode   string     `json:"short_code"`
		OriginalURL string     `json:"original_url"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.WriteError(w, apierr.NewValidation("invalid request payload"))
		return
	}

	url, err := h.service.CreateShortURL(req.ShortCode, req.OriginalURL, req.ExpiresAt)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(url)
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
