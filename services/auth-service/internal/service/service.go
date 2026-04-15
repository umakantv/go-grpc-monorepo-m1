package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	auth "github.com/yourorg/monorepo/gen/go/private/auth"
	"github.com/yourorg/monorepo/services/auth-service/internal/config"
	"github.com/yourorg/monorepo/services/auth-service/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/golang-jwt/jwt/v5"
)

// AuthService implements the AuthService gRPC service
type AuthService struct {
	auth.UnimplementedAuthServiceServer

	repo   *repository.Repository
	cfg    *config.JWTConfig
	logger *zap.Logger
}

// Claims represents JWT claims
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// NewAuthService creates a new AuthService
func NewAuthService(repo *repository.Repository, cfg *config.JWTConfig, logger *zap.Logger) *AuthService {
	return &AuthService{
		repo:   repo,
		cfg:    cfg,
		logger: logger.Named("auth-service"),
	}
}

// ValidateToken validates a JWT token and returns user info
func (s *AuthService) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	tokenString := req.GetToken()

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})

	if err != nil {
		return &auth.ValidateTokenResponse{
			Valid:        false,
			ErrorMessage: err.Error(),
		}, nil
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return &auth.ValidateTokenResponse{
			Valid:        false,
			ErrorMessage: "invalid token claims",
		}, nil
	}

	// Check if token is revoked
	storedToken, err := s.repo.GetToken(ctx, tokenString)
	if err != nil && !errors.Is(err, repository.ErrTokenNotFound) {
		s.logger.Error("failed to check token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to validate token")
	}

	if storedToken != nil && storedToken.Revoked {
		return &auth.ValidateTokenResponse{
			Valid:        false,
			ErrorMessage: "token has been revoked",
		}, nil
	}

	return &auth.ValidateTokenResponse{
		Valid: true,
		TokenInfo: &auth.TokenInfo{
			UserId:    claims.UserID,
			Email:     claims.Email,
			Roles:     claims.Roles,
			ExpiresAt: timestamppb.New(claims.ExpiresAt.Time),
		},
	}, nil
}

// GenerateToken generates a new JWT token for a user
func (s *AuthService) GenerateToken(ctx context.Context, req *auth.GenerateTokenRequest) (*auth.GenerateTokenResponse, error) {
	// Calculate expiration times
	accessTTL := s.cfg.AccessTokenTTL
	if req.GetExpiresInSeconds() > 0 {
		accessTTL = int(req.GetExpiresInSeconds())
	}

	now := time.Now()
	accessExpiry := now.Add(time.Duration(accessTTL) * time.Second)
	refreshExpiry := now.Add(time.Duration(s.cfg.RefreshTokenTTL) * time.Second)

	// Create access token
	accessClaims := Claims{
		UserID: req.GetUserId(),
		Email:  req.GetEmail(),
		Roles:  req.GetRoles(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    s.cfg.Issuer,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		s.logger.Error("failed to sign access token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	// Generate refresh token
	refreshTokenString, err := generateRandomString(32)
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	// Store refresh token in database
	refreshToken := &repository.Token{
		ID:        generateID(),
		UserID:    req.GetUserId(),
		Token:     refreshTokenString,
		Type:      "refresh",
		Revoked:   false,
		ExpiresAt: refreshExpiry,
	}

	if err := s.repo.StoreToken(ctx, refreshToken); err != nil {
		s.logger.Error("failed to store refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	s.logger.Info("token generated", zap.String("user_id", req.GetUserId()))

	return &auth.GenerateTokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    timestamppb.New(accessExpiry),
	}, nil
}

// RefreshToken refreshes an existing token
func (s *AuthService) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest) (*auth.RefreshTokenResponse, error) {
	refreshTokenString := req.GetRefreshToken()

	// Look up refresh token
	storedToken, err := s.repo.GetToken(ctx, refreshTokenString)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
		}
		s.logger.Error("failed to get refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	// Check if token is revoked or expired
	if storedToken.Revoked {
		return nil, status.Error(codes.Unauthenticated, "refresh token has been revoked")
	}
	if storedToken.ExpiresAt.Before(time.Now()) {
		return nil, status.Error(codes.Unauthenticated, "refresh token has expired")
	}

	// Generate new access token
	now := time.Now()
	accessExpiry := now.Add(time.Duration(s.cfg.AccessTokenTTL) * time.Second)

	accessClaims := Claims{
		UserID: storedToken.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    s.cfg.Issuer,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		s.logger.Error("failed to sign access token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	// Generate new refresh token
	newRefreshTokenString, err := generateRandomString(32)
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	// Revoke old refresh token
	if err := s.repo.RevokeToken(ctx, refreshTokenString); err != nil {
		s.logger.Error("failed to revoke old refresh token", zap.Error(err))
		// Continue anyway
	}

	// Store new refresh token
	refreshExpiry := now.Add(time.Duration(s.cfg.RefreshTokenTTL) * time.Second)
	newRefreshToken := &repository.Token{
		ID:        generateID(),
		UserID:    storedToken.UserID,
		Token:     newRefreshTokenString,
		Type:      "refresh",
		Revoked:   false,
		ExpiresAt: refreshExpiry,
	}

	if err := s.repo.StoreToken(ctx, newRefreshToken); err != nil {
		s.logger.Error("failed to store new refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	return &auth.RefreshTokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: newRefreshTokenString,
		ExpiresAt:    timestamppb.New(accessExpiry),
	}, nil
}

// RevokeToken revokes a token (for logout)
func (s *AuthService) RevokeToken(ctx context.Context, req *auth.RevokeTokenRequest) (*auth.RevokeTokenResponse, error) {
	tokenString := req.GetToken()

	// Try to revoke the token
	err := s.repo.RevokeToken(ctx, tokenString)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			// Token not found, but that's okay for logout
			return &auth.RevokeTokenResponse{Success: true}, nil
		}
		s.logger.Error("failed to revoke token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to revoke token")
	}

	s.logger.Info("token revoked")

	return &auth.RevokeTokenResponse{Success: true}, nil
}

// Helper functions

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}
