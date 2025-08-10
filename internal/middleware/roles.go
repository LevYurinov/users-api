package middleware

import (
	"go.uber.org/zap"
	"net/http"
)

const (
	RoleGuest  = "guest"
	RoleAdmin  = "admin"
	RoleEditor = "editor"
)

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log := LoggerFromContext(r.Context())

		role, ok := r.Context().Value("role").(string)
		if !ok || role != RoleAdmin {
			log.Error("invalid role",
				zap.String("component", "middleware"),
				zap.String("event", "role_checking"),
				zap.String("required_role", RoleAdmin),
				zap.String("got_role", role),
			)

			http.Error(w, "access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func EditorOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log := LoggerFromContext(r.Context())

		role, ok := r.Context().Value("role").(string)
		if !ok || role != RoleEditor {
			log.Error("invalid role",
				zap.String("component", "middleware"),
				zap.String("event", "role_checking"),
				zap.String("required_role", RoleAdmin),
				zap.String("got_role", role),
			)

			http.Error(w, "access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
