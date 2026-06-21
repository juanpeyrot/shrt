package web

import (
	"errors"
	"net/http"
	"strconv"

	"shrt/internal/apierr"
	"shrt/internal/auth"

	"github.com/go-chi/chi/v5"
)

func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())

	page := 1
	pageSize := 20

	result, err := h.linkService.ListLinks(claims.UserID, page, pageSize)
	if err != nil {
		http.Error(w, "Failed to load links", http.StatusInternalServerError)
		return
	}

	h.templates.Render(w, "dashboard", h.pageData(r, map[string]any{
		"Links":      result.Data,
		"Page":       result.Page,
		"PageSize":   result.PageSize,
		"TotalItems": result.TotalItems,
		"TotalPages": result.TotalPages,
		"BaseURL":    h.baseURL,
	}))
}

func (h *WebHandler) LinkTable(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	result, err := h.linkService.ListLinks(claims.UserID, page, 20)
	if err != nil {
		http.Error(w, "Failed to load links", http.StatusInternalServerError)
		return
	}

	h.templates.RenderPartial(w, "link_table", map[string]any{
		"Links":      result.Data,
		"Page":       result.Page,
		"PageSize":   result.PageSize,
		"TotalItems": result.TotalItems,
		"TotalPages": result.TotalPages,
		"BaseURL":    h.baseURL,
	})
}

func (h *WebHandler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	shortCode := chi.URLParam(r, "shortCode")

	if err := h.linkService.DeleteLink(claims.UserID, shortCode); err != nil {
		http.Error(w, "Failed to delete link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	shortCode := chi.URLParam(r, "shortCode")

	link, err := h.linkService.RetrieveLink(claims.UserID, shortCode)
	if err != nil {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	h.templates.RenderPartial(w, "link_edit", map[string]any{
		"Link":    link,
		"BaseURL": h.baseURL,
	})
}

func (h *WebHandler) UpdateLink(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	shortCode := chi.URLParam(r, "shortCode")
	newURL := r.FormValue("url")

	link, err := h.linkService.UpdateLink(claims.UserID, shortCode, newURL)
	if err != nil {
		errMsg := "Failed to update link"
		var appErr *apierr.AppError
		if errors.As(err, &appErr) {
			errMsg = appErr.Message
		}
		h.templates.RenderPartial(w, "link_edit", map[string]any{
			"Link":    link,
			"Error":   errMsg,
			"BaseURL": h.baseURL,
		})
		return
	}

	h.templates.RenderPartial(w, "link_row", map[string]any{
		"Link":    link,
		"BaseURL": h.baseURL,
	})
}

func (h *WebHandler) LinkRow(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	shortCode := chi.URLParam(r, "shortCode")

	link, err := h.linkService.RetrieveLink(claims.UserID, shortCode)
	if err != nil {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	h.templates.RenderPartial(w, "link_row", map[string]any{
		"Link":    link,
		"BaseURL": h.baseURL,
	})
}

func (h *WebHandler) LinkStats(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	shortCode := chi.URLParam(r, "shortCode")

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	link, err := h.linkService.RetrieveLink(claims.UserID, shortCode)
	if err != nil {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	stats, err := h.linkService.GetStats(claims.UserID, shortCode, days)
	if err != nil {
		http.Error(w, "Failed to load stats", http.StatusInternalServerError)
		return
	}

	var maxClicks int64
	for _, dc := range stats.ClicksPerDay {
		if dc.Clicks > maxClicks {
			maxClicks = dc.Clicks
		}
	}

	h.templates.RenderPartial(w, "link_stats", map[string]any{
		"Link":      link,
		"Stats":     stats,
		"MaxClicks": maxClicks,
		"BaseURL":   h.baseURL,
	})
}
