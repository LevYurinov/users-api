package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recoverer — middleware, которое перехватывает panic и возвращает 500
func Recoverer(logger *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {

			// defer-функция отработает даже при panic
			defer func() {
				err := recover()
				if err != nil {
					// Логгируем panic и stack trace
					logger.Printf("[PANIC RECOVERED] %v\n%s", err, debug.Stack())

					// Возвращаем 500 клиенту
					http.Error(wr, "Внутренняя ошибка сервера", http.StatusInternalServerError)
				}
			}()
			// Передаём выполнение дальше
			next.ServeHTTP(wr, req)
		})
	}
}
