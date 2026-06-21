package web

import (
	"net/http"

	"shrt/internal/auth"

	"github.com/go-chi/chi/v5"
)

func (h *WebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.ClaimsFromContext(r.Context()); ok {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	h.templates.Render(w, "login", h.pageData(r, nil))
}

func (h *WebHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	tokens, err := h.authService.LoginLocal(email, password)
	if err != nil {
		h.templates.Render(w, "login", h.pageData(r, map[string]any{
			"Error": "Invalid email or password.",
			"Email": email,
		}))
		return
	}

	h.setAuthCookies(w, tokens.AccessToken, tokens.RefreshToken)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (h *WebHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.ClaimsFromContext(r.Context()); ok {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	h.templates.Render(w, "register", h.pageData(r, nil))
}

func (h *WebHandler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	displayName := r.FormValue("display_name")

	tokens, err := h.authService.RegisterLocal(email, password, displayName)
	if err != nil {
		h.templates.Render(w, "register", h.pageData(r, map[string]any{
			"Error":       err.Error(),
			"Email":       email,
			"DisplayName": displayName,
		}))
		return
	}

	h.setAuthCookies(w, tokens.AccessToken, tokens.RefreshToken)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (h *WebHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.clearAuthCookies(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *WebHandler) WebOAuthStart(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if _, ok := h.providers.Get(provider); !ok {
		http.Error(w, "Unknown provider", http.StatusNotFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_web",
		Value:    "1",
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/api/auth/"+provider, http.StatusFound)
}

