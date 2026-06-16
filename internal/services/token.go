package services

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"shrt/internal/auth"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type tokenType string

const (
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

type TokenService struct {
	secret []byte
	repo   TokenRepository
}

func NewTokenService(secret []byte, repo TokenRepository) *TokenService {
	return &TokenService{secret: secret, repo: repo}
}

func (s *TokenService) IssueTokenPair(userID uuid.UUID) (auth.TokenPair, error) {
	accessToken, err := s.newAccessToken(userID)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("access token: %w", err)
	}

	refreshToken, jti, err := s.newRefreshToken(userID)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("refresh token: %w", err)
	}

	hash := hashRefreshToken(refreshToken)

	if err := s.repo.SaveRefreshToken(userID, hash, jti); err != nil {
		return auth.TokenPair{}, fmt.Errorf("save refresh token: %w", err)
	}

	return auth.TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *TokenService) RefreshTokens(rawRefreshToken string) (auth.TokenPair, error) {
	claims, err := s.parseRefreshToken(rawRefreshToken)
	if err != nil {
		return auth.TokenPair{}, ErrInvalidToken
	}

	storedHash, storedJTI, err := s.repo.GetRefreshToken(claims.UserID)
	if err != nil {
		return auth.TokenPair{}, ErrInvalidToken
	}

	if storedJTI != claims.ID {
		_ = s.repo.DeleteRefreshToken(claims.UserID)
		return auth.TokenPair{}, ErrTokenReuse
	}

	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashRefreshToken(rawRefreshToken))) != 1 {
		return auth.TokenPair{}, ErrInvalidToken
	}

	return s.IssueTokenPair(claims.UserID)
}

func (s *TokenService) Revoke(userID uuid.UUID) error {
	return s.repo.DeleteRefreshToken(userID)
}

func (s *TokenService) newAccessToken(userID uuid.UUID) (string, error) {
	claims := auth.Claims{
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

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
