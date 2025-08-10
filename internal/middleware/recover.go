package middleware

import (
	"go.uber.org/zap"
	"net/http"
	"runtime/debug"
)

// Recoverer — middleware, который перехватывает паники, логирует ошибку и возвращает HTTP 500
func Recoverer() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {

			log := LoggerFromContext(req.Context())

			defer func() {
				err := recover()
				if err != nil {
					log.Error(
						"panic recovered",
						zap.Any("error", err),
						zap.ByteString("stack", debug.Stack()),
					)

					http.Error(wr, "Внутренняя ошибка сервера", http.StatusInternalServerError)
				}
			}()
			// Передаём выполнение дальше
			next.ServeHTTP(wr, req)
		})
	}
}
