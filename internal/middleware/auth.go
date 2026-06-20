package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"shrt/internal/apierr"
	"shrt/internal/auth"
)

func Authenticate(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				apierr.WriteError(w, apierr.NewUnauthorized("missing token"))
				return
			}

			claims := &auth.Claims{}
			_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			})
			if err != nil {
				apierr.WriteError(w, apierr.NewUnauthorized("invalid token"))
				return
			}

			ctx := auth.WithClaims(r.Context(), claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuthenticate(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token != "" {
				claims := &auth.Claims{}
				_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return secret, nil
				})
				if err == nil {
					r = r.WithContext(auth.WithClaims(r.Context(), claims))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
