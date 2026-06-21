package web

import (
	"errors"
	"net/http"
	"time"

	"shrt/internal/apierr"
	"shrt/internal/auth"

	"github.com/google/uuid"
)

func (h *WebHandler) Home(w http.ResponseWriter, r *http.Request) {
	h.templates.Render(w, "home", h.pageData(r, nil))
}

func (h *WebHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	originalURL := r.FormValue("original_url")
	shortCode := r.FormValue("short_code")
	expiresAtStr := r.FormValue("expires_at")

	var expiresAt *time.Time
	if expiresAtStr != "" {
		t, err := time.Parse("2006-01-02T15:04", expiresAtStr)
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
		errMsg := "Something went wrong"
		var appErr *apierr.AppError
		if errors.As(err, &appErr) {
			errMsg = appErr.Message
		}
		h.templates.RenderPartial(w, "link_result", map[string]any{
			"Error": errMsg,
		})
		return
	}

	h.templates.RenderPartial(w, "link_result", map[string]any{
		"ShortURL":    h.baseURL + "/" + link.ShortCode,
		"OriginalURL": link.OriginalURL,
	})
}
