package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/akshaya-cp/golang_project/internal/auth"
	"github.com/akshaya-cp/golang_project/internal/cache"
	"github.com/akshaya-cp/golang_project/internal/model"
	"github.com/akshaya-cp/golang_project/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidRefresh     = errors.New("invalid or expired refresh token")
)

const (
	refreshKeyPrefix = "refresh:"
	profileKeyPrefix = "user:profile:"
	profileCacheTTL  = 5 * time.Minute
)

type AuthService struct {
	users      *repository.UserRepository
	tokens     *auth.TokenManager
	cache      *cache.Client
	refreshTTL time.Duration
}

func NewAuthService(users *repository.UserRepository, tokens *auth.TokenManager, c *cache.Client, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		users:      users,
		tokens:     tokens,
		cache:      c,
		refreshTTL: refreshTTL,
	}
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	User         model.UserResponse
}

func (s *AuthService) Signup(ctx context.Context, email, password, name string) (*AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(ctx, email, hash, name, "user")
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user)
}

// Refresh rotates a refresh token: the presented token is validated, deleted,
// and a fresh access/refresh pair is issued (refresh-token rotation).
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	if s.cache == nil {
		return nil, ErrInvalidRefresh
	}

	key := refreshKeyPrefix + refreshToken
	userIDStr, ok, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidRefresh
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidRefresh
	}

	// Rotate: invalidate the old refresh token immediately.
	_ = s.cache.Delete(ctx, key)

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}

	return s.issueTokens(ctx, user)
}

// Logout revokes a refresh token so it can no longer be used.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Delete(ctx, refreshKeyPrefix+refreshToken)
}

// GetProfile returns the user's public profile, using Redis as a cache-aside
// layer to avoid hitting Postgres on every authenticated request.
func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserResponse, error) {
	cacheKey := profileKeyPrefix + userID.String()

	if s.cache != nil {
		var cached model.UserResponse
		if hit, err := s.cache.GetJSON(ctx, cacheKey, &cached); err == nil && hit {
			return &cached, nil
		}
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := user.ToResponse()

	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, cacheKey, resp, profileCacheTTL)
	}

	return &resp, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *model.User) (*AuthResult, error) {
	accessToken, expiresAt, err := s.tokens.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if err := s.cache.Set(ctx, refreshKeyPrefix+refreshToken, user.ID.String(), s.refreshTTL); err != nil {
			return nil, fmt.Errorf("store refresh token: %w", err)
		}
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         user.ToResponse(),
	}, nil
}
