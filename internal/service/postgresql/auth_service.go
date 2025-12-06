package service

import (
	"errors"

	entity "home-market/internal/domain"
	repo "home-market/internal/repository/postgresql"
	"home-market/pkg"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrInactiveAccount     = errors.New("account is inactive")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrUserNotFound        = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
	ErrEmailTaken    = errors.New("email already taken")
)




type AuthService struct {
	userRepo      repo.UserRepository
	defaultRoleID uuid.UUID
}

func NewAuthService(userRepo repo.UserRepository, defaultRoleID uuid.UUID) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		defaultRoleID: defaultRoleID,
	}
}

// @Summary      User Login
// @Description  Authenticate a user with username and password, returns access and refresh tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  entity.LoginResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/login [post]
func (s *AuthService) Login(username, password string) (*entity.LoginResponse, error) {
	user, roleName, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrInactiveAccount
	}

	permissions, err := s.userRepo.GetPermissionsByRoleID(user.RoleID)
	if err != nil {
		return nil, err
	}

	tokenString, err := utils.GenerateToken(user, roleName, permissions)
	if err != nil {
		return nil, err
	}

	refresh, err := utils.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	resp := &entity.LoginResponse{
		Token:        tokenString,
		RefreshToken: refresh,
		User: entity.UserResp{
			ID:          user.ID,
			Username:    user.Username,
			FullName:    user.FullName,
			Role:        roleName,
			Permissions: permissions,
		},
	}

	return resp, nil
}

// @Summary      Register New User
// @Description  Creates a new user account with default 'buyer' role.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      201  {object}  entity.UserResp
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /auth/register [post]
func (s *AuthService) Register(input *entity.RegisterInput) (*entity.UserResp, error) {
	// cek username
	if u, _, _ := s.userRepo.GetByUsername(input.Username); u != nil {
		return nil, ErrUsernameTaken
	}

	// cek email
	if u, _ := s.userRepo.GetByEmail(input.Email); u != nil {
		return nil, ErrEmailTaken
	}

	// hash password
	hashed, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	// role buyer 
	roleID := s.defaultRoleID
	if roleID == uuid.Nil {
		return nil, errors.New("default role 'buyer' is not set")
	}

	user := &entity.User{
		ID:           uuid.New(),
		Username:     input.Username,
		Email:        input.Email,
		FullName:     input.FullName,
		PasswordHash: hashed,
		RoleID:       roleID, 
		IsActive:     true,
	}

	// simpan user
	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, err
	}

	// nama role buyer
	_, roleName, err := s.userRepo.GetByUsername(user.Username)
	if err != nil {
		roleName = "buyer" 
	}

	resp := &entity.UserResp{
		ID:          user.ID,
		Username:    user.Username,
		FullName:    user.FullName,
		Role:        roleName,     
		Permissions: []string{},   
	}

	return resp, nil
}

// @Summary      Refresh Access Token
// @Description  Exchanges a valid refresh token for a new access token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  entity.RefreshResponse
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func (s *AuthService) Refresh(refreshToken string) (string, error) {
	claims, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", ErrInvalidRefreshToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", ErrInvalidUserID
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	permissions, err := s.userRepo.GetPermissionsByRoleID(user.RoleID)
	if err != nil {
		return "", err
	}

	_, roleName, err := s.userRepo.GetByUsername(user.Username)
	if err != nil {
		return "", err
	}

	newToken, err := utils.GenerateToken(user, roleName, permissions)
	if err != nil {
		return "", err
	}

	return newToken, nil
}

// @Summary      Get User Profile
// @Description  Retrieves the current authenticated user's profile details.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  entity.UserResp
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /auth/profile [get]
func (s *AuthService) GetProfile(userID uuid.UUID) (*entity.UserResp, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	permissions, err := s.userRepo.GetPermissionsByRoleID(user.RoleID)
	if err != nil {
		return nil, err
	}

	_, roleName, err := s.userRepo.GetByUsername(user.Username)
	if err != nil {
		return nil, err
	}

	resp := &entity.UserResp{
		ID:          user.ID,
		Username:    user.Username,
		FullName:    user.FullName,
		Role:        roleName,
		Permissions: permissions,
	}

	return resp, nil
}
