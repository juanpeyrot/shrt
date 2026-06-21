package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shrt/internal/models"
	"shrt/internal/services"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) GetUserByID(id uuid.UUID) (models.User, error) {
	var u models.User
	err := r.db.QueryRow(context.Background(),
		`SELECT id, display_name, email, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.DisplayName, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, services.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("query user: %w", err)
	}
	return u, nil
}

func (r *AuthRepository) CreateUserWithAuthMethod(u models.User, m models.AuthMethod) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(),
		`INSERT INTO users (id, display_name, email, created_at)
		 VALUES ($1, $2, $3, $4)`,
		u.ID, u.DisplayName, u.Email, u.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		`INSERT INTO auth_methods (id, user_id, provider, provider_user_id, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ID, m.UserID, m.Provider, m.ProviderUserID, m.PasswordHash, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert auth_method: %w", err)
	}

	return tx.Commit(context.Background())
}

func (r *AuthRepository) GetUserByEmail(email string) (models.User, error) {
	var u models.User
	err := r.db.QueryRow(context.Background(),
		`SELECT id, display_name, email, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.DisplayName, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, services.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("query user: %w", err)
	}
	return u, nil
}

func (r *AuthRepository) GetLocalAuthByEmail(email string) (models.User, models.AuthMethod, error) {
	var u models.User
	var m models.AuthMethod
	err := r.db.QueryRow(context.Background(),
		`SELECT u.id, u.display_name, u.email, u.created_at,
		        m.id, m.user_id, m.provider, m.password_hash, m.created_at, m.updated_at
		 FROM users u
		 JOIN auth_methods m ON m.user_id = u.id AND m.provider = 'local'
		 WHERE u.email = $1`,
		email,
	).Scan(
		&u.ID, &u.DisplayName, &u.Email, &u.CreatedAt,
		&m.ID, &m.UserID, &m.Provider, &m.PasswordHash, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, models.AuthMethod{}, services.ErrUserNotFound
		}
		return models.User{}, models.AuthMethod{}, fmt.Errorf("query local auth: %w", err)
	}
	return u, m, nil
}

func (r *AuthRepository) GetUserByProviderID(provider, providerUserID string) (models.User, error) {
	var u models.User
	err := r.db.QueryRow(context.Background(),
		`SELECT u.id, u.display_name, u.email, u.created_at
		 FROM users u
		 JOIN auth_methods m ON m.user_id = u.id
		 WHERE m.provider = $1 AND m.provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&u.ID, &u.DisplayName, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, services.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("query user by provider id: %w", err)
	}
	return u, nil
}

func (r *AuthRepository) AddAuthMethod(m models.AuthMethod) error {
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO auth_methods (id, user_id, provider, provider_user_id, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ID, m.UserID, m.Provider, m.ProviderUserID, m.PasswordHash, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert auth_method: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetRefreshToken(userID uuid.UUID) (hash string, jwtID string, err error) {
	err = r.db.QueryRow(context.Background(),
		`SELECT refresh_token_hash, refresh_token_jwt_id
		 FROM auth_methods
		 WHERE user_id = $1
		 LIMIT 1`,
		userID,
	).Scan(&hash, &jwtID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", services.ErrUserNotFound
		}
		return "", "", fmt.Errorf("query refresh token: %w", err)
	}
	return hash, jwtID, nil
}

func (r *AuthRepository) SaveRefreshToken(userID uuid.UUID, hash string, jwtID string) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE auth_methods
		 SET refresh_token_hash = $2, refresh_token_jwt_id = $3, updated_at = NOW()
		 WHERE user_id = $1`,
		userID, hash, jwtID,
	)
	if err != nil {
		return fmt.Errorf("update refresh token: %w", err)
	}
	return nil
}

func (r *AuthRepository) DeleteRefreshToken(userID uuid.UUID) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE auth_methods
		 SET refresh_token_hash = NULL, refresh_token_jwt_id = NULL, updated_at = NOW()
		 WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("clear refresh token: %w", err)
	}
	return nil
}
