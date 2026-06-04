package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/akshaya-cp/golang_project/internal/auth"
	"github.com/akshaya-cp/golang_project/internal/model"
	"github.com/akshaya-cp/golang_project/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
)

type AuthService struct {
	users  *repository.UserRepository
	tokens *auth.TokenManager
}

func NewAuthService(users *repository.UserRepository, tokens *auth.TokenManager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

type AuthResult struct {
	AccessToken string
	ExpiresAt   time.Time
	User        model.UserResponse
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

	return s.issueToken(user)
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

	return s.issueToken(user)
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := user.ToResponse()
	return &resp, nil
}

func (s *AuthService) issueToken(user *model.User) (*AuthResult, error) {
	token, expiresAt, err := s.tokens.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken: token,
		ExpiresAt:   expiresAt,
		User:        user.ToResponse(),
	}, nil
}
