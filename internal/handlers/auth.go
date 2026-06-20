package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"shrt/internal/apierr"
	"shrt/internal/auth/oauth"
	"shrt/internal/services"
)

const oauthStateCookie = "oauth_state"

type AuthHandler struct {
	service   *services.AuthService
	providers oauth.Registry
}

func NewAuthHandler(service *services.AuthService, providers oauth.Registry) *AuthHandler {
	return &AuthHandler{service: service, providers: providers}
}

func (h *AuthHandler) RegisterLocal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.WriteError(w, apierr.NewValidation("invalid request payload"))
		return
	}

	tokens, err := h.service.RegisterLocal(req.Email, req.Password, req.DisplayName)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, tokens)
}

func (h *AuthHandler) LoginLocal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.WriteError(w, apierr.NewValidation("invalid request payload"))
		return
	}

	tokens, err := h.service.LoginLocal(req.Email, req.Password)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) OAuthRedirect(w http.ResponseWriter, r *http.Request) {
	provider, ok := h.providers.Get(chi.URLParam(r, "provider"))
	if !ok {
		apierr.WriteError(w, apierr.NewNotFound("unknown provider"))
		return
	}

	state, err := randomState()
	if err != nil {
		apierr.WriteError(w, apierr.NewInternal("failed to generate state", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(10 * time.Minute),
	})

	http.Redirect(w, r, provider.AuthURL(state), http.StatusFound)
}

func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "provider")
	provider, ok := h.providers.Get(name)
	if !ok {
		apierr.WriteError(w, apierr.NewNotFound("unknown provider"))
		return
	}

	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" || cookie.Value != r.URL.Query().Get("state") {
		apierr.WriteError(w, apierr.NewUnauthorized("invalid oauth state"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		apierr.WriteError(w, apierr.NewValidation("missing authorization code"))
		return
	}

	profile, err := provider.Exchange(r.Context(), code)
	if err != nil {
		apierr.WriteError(w, apierr.NewUnauthorized("oauth exchange failed"))
		return
	}

	tokens, err := h.service.AuthenticateOAuth(name, profile)
	if err != nil {
		apierr.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
