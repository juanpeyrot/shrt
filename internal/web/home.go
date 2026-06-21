package web

import (
	"net/http"
	"time"

	"shrt/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *WebHandler) Home(w http.ResponseWriter, r *http.Request) {
	h.templates.Render(w, "home", h.pageData(r, time.Now().Format("2006-01-02T15:04")))
}

func (h *WebHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.templates.Render(w, "notfound", h.pageData(r, nil))
}

func (h *WebHandler) RedirectShortCode(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")

	originalURL, err := h.linkService.GetByShortCode(shortCode, r.Referer())
	if err != nil {
		h.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func (h *WebHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	originalURL := r.FormValue("original_url")
	shortCode := r.FormValue("short_code")
	expiresAtStr := r.FormValue("expires_at")

	var expiresAt *time.Time
	if expiresAtStr != "" {
		t, err := time.ParseInLocation("2006-01-02T15:04", expiresAtStr, time.Local)
		if err == nil {
			expiresAt = &t
		}
	}

	var userID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		userID = &claims.UserID
	}

	link, err := h.linkService.CreateShortURL(userID, shortCode, originalURL, expiresAt)
	if err != nil {
		h.templates.RenderPartial(w, "link_result", map[string]any{
			"Error": friendlyError(err),
		})
		return
	}

	h.templates.RenderPartial(w, "link_result", map[string]any{
		"ShortURL":    h.baseURL + "/" + link.ShortCode,
		"OriginalURL": link.OriginalURL,
	})
}
