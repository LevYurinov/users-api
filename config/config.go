package config

import "time"

// переменные для создания токенов
var (
	JWTSecret       = []byte("super-secret-key")
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 1200 * time.Hour
)
