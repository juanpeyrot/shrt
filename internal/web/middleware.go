package web

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"shrt/internal/auth"
	"shrt/internal/services"
)

func WebOptionalAuth(jwtSecret []byte, tokenSvc *services.TokenService, authSvc *services.AuthService, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := parseCookieToken(r, jwtSecret)
			if !ok {
				claims, ok = tryRefresh(w, r, jwtSecret, tokenSvc, secure)
			}
			if ok {
				ctx := auth.WithClaims(r.Context(), claims)
				if user, err := authSvc.GetUser(claims.UserID); err == nil {
					ctx = WithUser(ctx, &user)
				}
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func WebAuthenticate(jwtSecret []byte, tokenSvc *services.TokenService, authSvc *services.AuthService, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := parseCookieToken(r, jwtSecret)
			if !ok {
				claims, ok = tryRefresh(w, r, jwtSecret, tokenSvc, secure)
			}
			if !ok {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			ctx := auth.WithClaims(r.Context(), claims)
			if user, err := authSvc.GetUser(claims.UserID); err == nil {
				ctx = WithUser(ctx, &user)
			}
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func parseCookieToken(r *http.Request, secret []byte) (*auth.Claims, bool) {
	cookie, err := r.Cookie("access_token")
	if err != nil || cookie.Value == "" {
		return nil, false
	}
	claims := &auth.Claims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil {
		return nil, false
	}
	return claims, true
}

func tryRefresh(w http.ResponseWriter, r *http.Request, secret []byte, tokenSvc *services.TokenService, secure bool) (*auth.Claims, bool) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		return nil, false
	}
	pair, err := tokenSvc.RefreshTokens(cookie.Value)
	if err != nil {
		return nil, false
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    pair.AccessToken,
		Path:     "/",
		MaxAge:   900,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    pair.RefreshToken,
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	claims := &auth.Claims{}
	jwt.ParseWithClaims(pair.AccessToken, claims, func(t *jwt.Token) (any, error) {
		return secret, nil
	})
	return claims, true
}
