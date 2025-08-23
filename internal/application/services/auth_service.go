package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/config"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// Claims represents the JWT claims
type Claims struct {
	UserID string            `json:"user_id"`
	Email  string            `json:"email"`
	Role   entities.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// AuthService handles authentication operations
type AuthService struct {
	userRepo  ports.UserRepository
	authRepo  ports.AuthRepository
	jwtConfig config.JWTConfig
	logger    *logger.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo ports.UserRepository, authRepo ports.AuthRepository, jwtConfig config.JWTConfig, logger *logger.Logger) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		authRepo:  authRepo,
		jwtConfig: jwtConfig,
		logger:    logger,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req ports.RegisterRequest) (*ports.AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	existingUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with username %s already exists", req.Username)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &entities.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User registered successfully", "user_id", createdUser.ID, "email", createdUser.Email)

	// Generate tokens
	accessToken, err := s.generateAccessToken(createdUser)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, createdUser.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Remove password hash from response
	createdUser.PasswordHash = ""

	return &ports.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtConfig.ExpiresIn.Seconds()),
		User:         createdUser,
	}, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req ports.LoginRequest) (*ports.AuthResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Warn("Login attempt with non-existent email", "email", req.Email)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		s.logger.Warn("Login attempt with inactive account", "email", req.Email, "user_id", user.ID)
		return nil, fmt.Errorf("account is inactive")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		s.logger.Warn("Login attempt with invalid password", "email", req.Email, "user_id", user.ID)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login time
	err = s.userRepo.UpdateLastLogin(ctx, user.ID, time.Now())
	if err != nil {
		s.logger.Warn("Failed to update last login time", "error", err, "user_id", user.ID)
	}

	s.logger.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return &ports.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtConfig.ExpiresIn.Seconds()),
		User:         user,
	}, nil
}

// RefreshToken generates a new access token using refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*ports.AuthResponse, error) {
	// Hash the refresh token to match stored value
	hasher := sha256.New()
	hasher.Write([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Get refresh token from database
	storedToken, err := s.authRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check if token is expired
	if storedToken.IsExpired() {
		return nil, fmt.Errorf("refresh token expired")
	}

	// Check if token is revoked
	if storedToken.IsRevoked() {
		return nil, fmt.Errorf("refresh token revoked")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Revoke old refresh token
	err = s.authRepo.RevokeRefreshToken(ctx, tokenHash)
	if err != nil {
		s.logger.Warn("Failed to revoke old refresh token", "error", err, "user_id", user.ID)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return &ports.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtConfig.ExpiresIn.Seconds()),
		User:         user,
	}, nil
}

// Logout revokes all refresh tokens for a user
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	err := s.authRepo.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	s.logger.Info("User logged out successfully", "user_id", userID)
	return nil
}

// ValidateToken validates a JWT token and returns claims
func (s *AuthService) ValidateToken(tokenString string) (*ports.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtConfig.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &ports.Claims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

func (s *AuthService) generateAccessToken(user *entities.User) (string, error) {
	claims := &Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtConfig.ExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.jwtConfig.Issuer,
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) generateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	token := hex.EncodeToString(tokenBytes)

	// Hash token for storage
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Store in database
	expiresAt := time.Now().Add(s.jwtConfig.RefreshExpiresIn)
	err = s.authRepo.CreateRefreshToken(ctx, userID, tokenHash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return token, nil
}
