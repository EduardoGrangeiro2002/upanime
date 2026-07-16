package handler

import (
	"context"
	"net/http"

	"upanime/api/auth"
	"upanime/api/store"
)

type contextKey string

const userEmailKey contextKey = "authEmail"

func RequireAuth(service *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				writeJSONError(w, "não autenticado", http.StatusUnauthorized)
				return
			}

			email, valid := service.VerifySession(cookie.Value)
			if !valid {
				writeJSONError(w, "sessão inválida ou expirada", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userEmailKey, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserEmail(ctx context.Context) string {
	email, _ := ctx.Value(userEmailKey).(string)
	return email
}

func RateLimitAuth(codes *auth.CodeStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := codes.AllowIP(r.Context(), clientIP(r))
			if err == nil && !allowed {
				writeJSONError(w, "muitas tentativas, aguarde alguns minutos", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin(users store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := users.GetByEmail(r.Context(), UserEmail(r.Context()))
			if err != nil || !user.IsAdmin {
				writeJSONError(w, "acesso restrito a administradores", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
