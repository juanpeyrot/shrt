package web

import (
	"context"
	"errors"
	"net/http"

	"shrt/internal/apierr"
	"shrt/internal/auth/oauth"
	"shrt/internal/models"
	"shrt/internal/services"
)

type userContextKey struct{}

type WebHandler struct {
	linkService  *services.LinkService
	authService  *services.AuthService
	tokenService *services.TokenService
	providers    oauth.Registry
	templates    *TemplateRegistry
	jwtSecret    []byte
	baseURL      string
	secureCookie bool
}

func NewWebHandler(
	linkSvc *services.LinkService,
	authSvc *services.AuthService,
	tokenSvc *services.TokenService,
	providers oauth.Registry,
	tmpl *TemplateRegistry,
	jwtSecret []byte,
	baseURL string,
	secureCookie bool,
) *WebHandler {
	return &WebHandler{
		linkService:  linkSvc,
		authService:  authSvc,
		tokenService: tokenSvc,
		providers:    providers,
		templates:    tmpl,
		jwtSecret:    jwtSecret,
		baseURL:      baseURL,
		secureCookie: secureCookie,
	}
}

type PageData struct {
	User    *models.User
	Content any
	BaseURL string
}

func (h *WebHandler) pageData(r *http.Request, content any) PageData {
	user, _ := UserFromContext(r.Context())
	return PageData{
		User:    user,
		Content: content,
		BaseURL: h.baseURL,
	}
}

func WithUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, u)
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userContextKey{}).(*models.User)
	return u, ok
}

func (h *WebHandler) setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   900,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

var userFriendlyErrors = map[string]string{
	// URL
	"url is required":                "Please enter a URL to shorten.",
	"url is invalid":                 "Please enter a valid URL (e.g. https://example.com).",
	"url is too long":                "The URL is too long. Maximum 2048 characters.",
	"url must start with http or https": "The URL must start with http:// or https://.",
	// Alias
	"short_code already in use":      "This custom alias is already taken. Try a different one.",
	"short_code must be 3-16 characters and can only contain letters, numbers, underscores, and hyphens": "Custom alias must be 3–16 characters long, using only letters, numbers, hyphens or underscores.",
	// Expiration
	"expires_at must be in the future": "The expiration date must be in the future.",
	// Auth
	"email already registered":       "This email is already registered. Try logging in.",
	"email is required":              "Please enter your email address.",
	"password must be at least 8 characters": "Password must be at least 8 characters.",
	"display_name must be at least 3 characters": "Display name must be at least 3 characters.",
	"display_name must be between 3 and 50 characters": "Display name must be between 3 and 50 characters.",
	"password must be between 8 and 128 characters": "Password must be between 8 and 128 characters.",
	"invalid credentials":            "Invalid email or password.",
	// Links
	"original_url is required":       "Please enter a URL to shorten.",
	"you don't own this short URL":   "You don't have permission to access this link.",
	"short URL not found":            "This link doesn't exist or has expired.",
}

func friendlyError(err error) string {
	var appErr *apierr.AppError
	if errors.As(err, &appErr) {
		if msg, ok := userFriendlyErrors[appErr.Message]; ok {
			return msg
		}
	}
	return "Something went wrong. Please try again."
}

func (h *WebHandler) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}
