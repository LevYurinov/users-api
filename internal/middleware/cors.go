package middleware

import (
	"github.com/rs/cors"
	"net/http"
)

func Cors() func(handler http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"https://example.com",
			"https://anotherdomain.com"},
		AllowedMethods:   []string{http.MethodGet, "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}).Handler
}
