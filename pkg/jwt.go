package utils

import (
	"errors"
	"os"
	"time"
	"home-market/internal/config"
	entity "home-market/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(user *entity.User, roleName string, permissions []string) (string, error) {
	jwtCfg := config.LoadJWT()

	claims := &entity.JWTClaims{
		UserID:      user.ID,
		RoleID:      user.RoleID,
		RoleName:    roleName,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(jwtCfg.TTLHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "student-performance-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtCfg.Secret)
}

func ValidateToken(tokenString string) (*entity.JWTClaims, error) {
	jwtCfg := config.LoadJWT()
	token, err := jwt.ParseWithClaims(tokenString, &entity.JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtCfg.Secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*entity.JWTClaims); ok && token.Valid {
    return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}



func GenerateRefreshToken(user *entity.User) (string, error) {
	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}

	expiration := time.Now().Add(7 * 24 * time.Hour)

	claims := &entity.RefreshClaims{
		UserID: user.ID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			Issuer:    "student-performance-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateRefreshToken(t string) (*entity.RefreshClaims, error) {
	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}

	token, err := jwt.ParseWithClaims(
		t,
		&entity.RefreshClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*entity.RefreshClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	return claims, nil
}
