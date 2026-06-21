package web

import (
	"context"
	"net/http"

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
