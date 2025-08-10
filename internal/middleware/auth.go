package middleware

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"net/http"
	"pet/config"
	"strings"
)

// contextKey — уникальный собственный тип для ключа контекста,
// чтобы безопасно использовать контекст без риска "переписать" чужие значения
type contextKey string

const userIDKey contextKey = "userID"

// Auth — middleware-функция для аутентификации по заголовку Authorization
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log := LoggerFromContext(r.Context())

		authHeader := r.Header.Get("Authorization")

		// Простейшая проверка на наличие токена в формате Bearer ...
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Error("missing or invalid token",
				zap.String("component", "middleware"),
				zap.String("event", "auth"),
			)

			http.Error(w, "access denied", http.StatusUnauthorized)
			return
		}

		// удаляет Bearer из записи с токеном
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// надо распарсить и проверить только access-токен
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {

			// Проверка, что используется правильный метод подписи
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return config.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			log.Error("processing token error",
				zap.Error(err),
				zap.String("component", "middleware"),
				zap.String("event", "auth"),
			)

			http.Error(w, "access denied", http.StatusUnauthorized)
			return
		}

		sub, ok := claims["sub"] // subject — ID пользователя (обязательное поле)
		if !ok {
			log.Error("claims token error",
				zap.String("component", "middleware"),
				zap.String("event", "auth"),
			)

			http.Error(w, "access denied", http.StatusUnauthorized)
			return
		}

		//TODO: переделать ID на тип "строка"

		userIDFloat, ok := sub.(float64)
		if !ok {
			log.Error("invalid sub-field",
				zap.String("component", "middleware"),
				zap.String("event", "auth"),
			)
			http.Error(w, "access denied", http.StatusUnauthorized)
			return
		}

		role, _ := claims["role"]

		// Добавляем userID в context
		ctx := context.WithValue(r.Context(), userIDKey, int(userIDFloat))
		ctx = context.WithValue(ctx, "role", role)

		// Передаём дальше с новым контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext — извлекает userID из context.Context
func GetUserIDFromContext(r *http.Request) (int, bool) {
	val := r.Context().Value(userIDKey)
	userID, ok := val.(int)
	return userID, ok
}
