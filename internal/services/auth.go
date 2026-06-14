package services

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"shrt/internal/apierr"
	"shrt/internal/auth"
	"shrt/internal/models"
)

type AuthRepository interface {
	CreateUser(u models.User) error
	GetUserByEmail(email string) (models.User, error)
}

type TokenIssuer interface {
	IssueTokenPair(userID uuid.UUID) (auth.TokenPair, error)
}

type AuthService struct {
	repo         AuthRepository
	tokenService TokenIssuer
}

func NewAuthService(repo AuthRepository, tokenService TokenIssuer) *AuthService {
	return &AuthService{repo: repo, tokenService: tokenService}
}

func (s *AuthService) RegisterLocal(email, password, displayName string) (auth.TokenPair, error) {
	if len(displayName) < 3 {
		return auth.TokenPair{}, apierr.NewValidation("display_name must be at least 3 characters")
	}
	if email == "" {
		return auth.TokenPair{}, apierr.NewValidation("email is required")
	}
	if len(password) < 8 {
		return auth.TokenPair{}, apierr.NewValidation("password must be at least 8 characters")
	}

	_, err := s.repo.GetUserByEmail(email)
	if err == nil {
		return auth.TokenPair{}, apierr.NewConflict("email already registered")
	}
	if !errors.Is(err, ErrUserNotFound) {
		return auth.TokenPair{}, apierr.NewInternal("failed to check email", err)
	}

	user := models.User{
		ID:          uuid.New(),
		DisplayName: displayName,
		Email:       email,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateUser(user); err != nil {
		return auth.TokenPair{}, apierr.NewInternal("failed to create user", err)
	}

	return s.tokenService.IssueTokenPair(user.ID)
}
