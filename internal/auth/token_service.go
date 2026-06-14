package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	bcryptCost      = 12
)

type tokenType string

const (
	tokenTypeAccess  tokenType = "access"
	tokenTypeRefresh tokenType = "refresh"
)

type refreshClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	TokenType tokenType `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenRepository interface {
	GetRefreshToken(userID uuid.UUID) (hash string, jwtID string, err error)
	SaveRefreshToken(userID uuid.UUID, hash string, jwtID string) error
	DeleteRefreshToken(userID uuid.UUID) error
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenReuse = errors.New("refresh token reuse detected")
)

type TokenService struct {
	secret []byte
	repo   TokenRepository
}

func NewTokenService(secret []byte, repo TokenRepository) *TokenService {
	return &TokenService{secret: secret, repo: repo}
}

func (s *TokenService) IssueTokenPair(userID uuid.UUID) (TokenPair, error) {
	accessToken, err := s.newAccessToken(userID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("access token: %w", err)
	}

	refreshToken, jti, err := s.newRefreshToken(userID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("refresh token: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcryptCost)
	if err != nil {
		return TokenPair{}, fmt.Errorf("hash refresh token: %w", err)
	}

	if err := s.repo.SaveRefreshToken(userID, string(hash), jti); err != nil {
		return TokenPair{}, fmt.Errorf("save refresh token: %w", err)
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *TokenService) RefreshTokens(rawRefreshToken string) (TokenPair, error) {
	claims, err := s.parseRefreshToken(rawRefreshToken)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}

	storedHash, storedJTI, err := s.repo.GetRefreshToken(claims.UserID)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}

	if storedJTI != claims.ID {
		_ = s.repo.DeleteRefreshToken(claims.UserID)
		return TokenPair{}, ErrTokenReuse
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(rawRefreshToken)); err != nil {
		return TokenPair{}, ErrInvalidToken
	}

	return s.IssueTokenPair(claims.UserID)
}

func (s *TokenService) Revoke(userID uuid.UUID) error {
	return s.repo.DeleteRefreshToken(userID)
}

func (s *TokenService) newAccessToken(userID uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *TokenService) newRefreshToken(userID uuid.UUID) (token string, jti string, err error) {
	jti = uuid.NewString()
	claims := refreshClaims{
		UserID:    userID,
		TokenType: tokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenTTL)),
		},
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	return token, jti, err
}

func (s *TokenService) parseRefreshToken(raw string) (*refreshClaims, error) {
	claims := &refreshClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims.TokenType != tokenTypeRefresh {
		return nil, errors.New("wrong token type")
	}
	return claims, nil
}
