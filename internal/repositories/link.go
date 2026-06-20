package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"shrt/internal/models"
	"shrt/internal/services"
)

type LinkRepository struct {
	db *pgxpool.Pool
}

func NewLinkRepository(db *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{db: db}
}

func (r *LinkRepository) CreateShortURL(url models.ShortURL) error {
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO links (id, user_id, short_code, original_url, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		url.ID, url.UserID, url.ShortCode, url.OriginalURL, url.CreatedAt, url.ExpiresAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return services.ErrDuplicateShortCode
		}
		return fmt.Errorf("db insert: %w", err)
	}
	return nil
}

func (r *LinkRepository) GetByShortCode(shortCode string) (string, error) {
	var originalURL string
	err := r.db.QueryRow(context.Background(),
		`SELECT original_url FROM links
		 WHERE short_code = $1
		   AND deleted_at IS NULL
		   AND (expires_at IS NULL OR expires_at > NOW())`,
		shortCode,
	).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", services.ErrShortCodeNotFound
		}
		return "", fmt.Errorf("db query: %w", err)
	}
	return originalURL, nil
}

func (r *LinkRepository) GetLinkByShortCode(shortCode string) (models.ShortURL, error) {
	var url models.ShortURL
	err := r.db.QueryRow(context.Background(),
		`SELECT id, user_id, short_code, original_url, created_at, expires_at, deleted_at, click_count
		 FROM links
		 WHERE short_code = $1
		   AND deleted_at IS NULL
		   AND (expires_at IS NULL OR expires_at > NOW())`,
		shortCode,
	).Scan(&url.ID, &url.UserID, &url.ShortCode, &url.OriginalURL, &url.CreatedAt, &url.ExpiresAt, &url.DeletedAt, &url.ClickCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ShortURL{}, services.ErrShortCodeNotFound
		}
		return models.ShortURL{}, fmt.Errorf("db query: %w", err)
	}
	return url, nil
}

func (r *LinkRepository) AddClick(shortCode string) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE links
		 SET click_count = click_count + 1
		 WHERE short_code = $1`,
		shortCode,
	)
	if err != nil {
		return fmt.Errorf("db update: %w", err)
	}
	return nil
}