package config

import (
	"os"
	"strconv"
)

type JWTConfig struct {
	Secret []byte
	TTLHours int
}

func LoadJWT() JWTConfig {
	secret := os.Getenv("JWT_SECRET")
	ttlStr := os.Getenv("JWT_TTL_HOURS")
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil || ttl <= 0 {
		ttl = 24
	}
	return JWTConfig{Secret: []byte(secret), TTLHours: ttl}
}
