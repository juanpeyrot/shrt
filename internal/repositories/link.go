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
		`INSERT INTO links (id, short_code, original_url, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		url.ID, url.ShortCode, url.OriginalURL, url.CreatedAt, url.ExpiresAt,
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
