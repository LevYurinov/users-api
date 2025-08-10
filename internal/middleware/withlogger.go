package middleware

import (
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
)

// ctxKeyLogger - неэкспортируемый ключ для передачи логгера в контекст
type ctxKeyLogger struct{}

// WithLogger - middleware-функция, которая кладет context в запрос и передает его дальше
// Она передает ID для трассировки, метод, путь и IP-адрес клиента, но с портом!
func WithLogger(baseLog *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {

			reqID := generateRequestID(req)

			reqLogger := baseLog.With(
				zap.String("trace.id", reqID),
				zap.String("http.method", req.Method),
				zap.String("url.path", req.URL.Path),
				zap.String("client.address", req.RemoteAddr),
			)

			ctx := context.WithValue(req.Context(), ctxKeyLogger{}, reqLogger)

			next.ServeHTTP(wr, req.WithContext(ctx))
		})
	}
}

func generateRequestID(req *http.Request) string {
	reqID := req.Header.Get("X-Request-ID")
	if reqID != "" {
		return reqID
	}
	return uuid.New().String()
}

func LoggerFromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(ctxKeyLogger{}).(*zap.Logger)
	if ok {
		return logger
	}
	return zap.NewNop() // пустой логгер, чтобы не паниковать
}
