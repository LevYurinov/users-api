package middleware

import "net/http"

const (
	RoleAdmin  = "admin"
	RoleUser   = "user"
	RoleEditor = "editor"
)

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("role").(string)
		if !ok || role != RoleAdmin {
			http.Error(w, "доступ запрещен, нужны повышенные права", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func EditorOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("role").(string)
		if !ok || role != RoleEditor {
			http.Error(w, "доступ запрещен, нужны повышенные права", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
