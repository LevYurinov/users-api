package middleware

import "net/http"

// DefaultHeaders возвращает middleware, которое устанавливает заголовки по умолчанию
func DefaultHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Устанавливаем системные заголовки безопасности и поведения по умолчанию
			w.Header().Set("Content-Type", "application/json")  // Все ответы — JSON
			w.Header().Set("X-Frame-Options", "DENY")           // Защита от Clickjacking
			w.Header().Set("X-Content-Type-Options", "nosniff") // Отключить MIME-sniffing
			w.Header().Set("Referrer-Policy", "no-referrer")    // Не отправлять Referer
			w.Header().Set("X-XSS-Protection", "1; mode=block") // Защита от XSS (старый, но безопасный флаг)
			w.Header().Set("Cache-Control", "no-store")         // Не кэшировать ответы

			// Передаём управление следующему хендлеру
			next.ServeHTTP(w, r)
		})
	}
}
