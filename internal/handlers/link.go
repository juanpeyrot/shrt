package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

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
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateShortURL(req.ShortCode, req.OriginalURL, req.ExpiresAt); err != nil {
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")

	originalURL, err := h.service.GetByShortCode(shortCode)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}
