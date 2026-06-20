package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"shrt/internal/apierr"
	"shrt/internal/auth"
	"shrt/internal/auth/oauth"
	"shrt/internal/models"
)

type AuthRepository interface {
	GetUserByEmail(email string) (models.User, error)
	GetLocalAuthByEmail(email string) (models.User, models.AuthMethod, error)
	GetUserByProviderID(provider, providerUserID string) (models.User, error)
	CreateUserWithAuthMethod(u models.User, m models.AuthMethod) error
	AddAuthMethod(m models.AuthMethod) error
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

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return auth.TokenPair{}, apierr.NewInternal("failed to hash password", err)
	}

	now := time.Now()
	user := models.User{
		ID:          uuid.New(),
		DisplayName: displayName,
		Email:       email,
		CreatedAt:   now,
	}
	hashStr := string(hash)
	authMethod := models.AuthMethod{
		ID:           uuid.New(),
		UserID:       user.ID,
		Provider:     "local",
		PasswordHash: &hashStr,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateUserWithAuthMethod(user, authMethod); err != nil {
		return auth.TokenPair{}, apierr.NewInternal("failed to create user", err)
	}

	return s.tokenService.IssueTokenPair(user.ID)
}

func (s *AuthService) LoginLocal(email, password string) (auth.TokenPair, error) {
	user, method, err := s.repo.GetLocalAuthByEmail(email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return auth.TokenPair{}, apierr.NewUnauthorized("invalid credentials")
		}
		return auth.TokenPair{}, apierr.NewInternal("failed to look up user", err)
	}

	if method.PasswordHash == nil {
		return auth.TokenPair{}, apierr.NewUnauthorized("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*method.PasswordHash), []byte(password)); err != nil {
		return auth.TokenPair{}, apierr.NewUnauthorized("invalid credentials")
	}

	return s.tokenService.IssueTokenPair(user.ID)
}

func (s *AuthService) AuthenticateOAuth(provider string, profile oauth.OAuthProfile) (auth.TokenPair, error) {
	if profile.ProviderUserID == "" {
		return auth.TokenPair{}, apierr.NewValidation("provider did not return a user id")
	}

	user, err := s.repo.GetUserByProviderID(provider, profile.ProviderUserID)
	if err == nil {
		return s.tokenService.IssueTokenPair(user.ID)
	}
	if !errors.Is(err, ErrUserNotFound) {
		return auth.TokenPair{}, apierr.NewInternal("failed to look up provider identity", err)
	}

	now := time.Now()
	method := models.AuthMethod{
		ID:             uuid.New(),
		Provider:       provider,
		ProviderUserID: profile.ProviderUserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if profile.EmailVerified && profile.Email != "" {
		existing, err := s.repo.GetUserByEmail(profile.Email)
		if err == nil {
			method.UserID = existing.ID
			if err := s.repo.AddAuthMethod(method); err != nil {
				return auth.TokenPair{}, apierr.NewInternal("failed to link auth method", err)
			}
			return s.tokenService.IssueTokenPair(existing.ID)
		}
		if !errors.Is(err, ErrUserNotFound) {
			return auth.TokenPair{}, apierr.NewInternal("failed to check email", err)
		}
	}

	displayName := profile.DisplayName
	if len(displayName) < 3 {
		displayName = provider + "_user"
	}
	newUser := models.User{
		ID:          uuid.New(),
		DisplayName: displayName,
		Email:       profile.Email,
		CreatedAt:   now,
	}
	method.UserID = newUser.ID
	if err := s.repo.CreateUserWithAuthMethod(newUser, method); err != nil {
		return auth.TokenPair{}, apierr.NewInternal("failed to create user", err)
	}

	return s.tokenService.IssueTokenPair(newUser.ID)
}
