package entity

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	RoleID      uuid.UUID `json:"role_id"`
	RoleName    string    `json:"role_name"`
	Permissions []string  `json:"permissions,omitempty"` 
	
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

type RefreshResponse struct {
    Token string `json:"token"`
}

type LoginInput struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserResp `json:"user"`
}

type UserResp struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	FullName    string    `json:"fullName"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
}

type RegisterInput struct {
	Username string
	Email    string
	FullName string
	Password string
}