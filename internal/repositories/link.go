package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *LinkRepository) UpdateOriginalURL(shortCode, originalURL string) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE links SET original_url = $1 WHERE short_code = $2`,
		originalURL, shortCode,
	)
	if err != nil {
		return fmt.Errorf("db update: %w", err)
	}
	return nil
}

func (r *LinkRepository) SoftDelete(shortCode string) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE links SET deleted_at = NOW() WHERE short_code = $1`,
		shortCode,
	)
	if err != nil {
		return fmt.Errorf("db update: %w", err)
	}
	return nil
}

func (r *LinkRepository) ListByUserID(userID fmt.Stringer, limit int, offset int) ([]models.ShortURL, int, error) {
	var total int
	err := r.db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM links WHERE user_id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("db count: %w", err)
	}

	rows, err := r.db.Query(context.Background(),
		`SELECT id, user_id, short_code, original_url, created_at, expires_at, deleted_at, click_count
		 FROM links
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("db query: %w", err)
	}
	defer rows.Close()

	var links []models.ShortURL
	for rows.Next() {
		var link models.ShortURL
		if err := rows.Scan(&link.ID, &link.UserID, &link.ShortCode, &link.OriginalURL, &link.CreatedAt, &link.ExpiresAt, &link.DeletedAt, &link.ClickCount); err != nil {
			return nil, 0, fmt.Errorf("db scan: %w", err)
		}
		links = append(links, link)
	}

	return links, total, nil
}

func (r *LinkRepository) AddClick(linkID fmt.Stringer, referer string) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(context.Background(),
		`INSERT INTO link_clicks (link_id, referer) VALUES ($1, $2)`,
		linkID, referer,
	); err != nil {
		return fmt.Errorf("db insert click: %w", err)
	}

	if _, err := tx.Exec(context.Background(),
		`UPDATE links SET click_count = click_count + 1 WHERE id = $1`,
		linkID,
	); err != nil {
		return fmt.Errorf("db update click_count: %w", err)
	}

	return tx.Commit(context.Background())
}

func (r *LinkRepository) GetStats(linkID fmt.Stringer, days int) (models.LinkStats, error) {
	var stats models.LinkStats

	err := r.db.QueryRow(context.Background(),
		`SELECT COUNT(*), MAX(clicked_at)
		 FROM link_clicks WHERE link_id = $1`,
		linkID,
	).Scan(&stats.TotalClicks, &stats.LastClickAt)
	if err != nil {
		return models.LinkStats{}, fmt.Errorf("db query stats: %w", err)
	}

	rows, err := r.db.Query(context.Background(),
		`SELECT clicked_at::date AS day, COUNT(*)
		 FROM link_clicks
		 WHERE link_id = $1 AND clicked_at >= NOW() - make_interval(days => $2)
		 GROUP BY day ORDER BY day`,
		linkID, days,
	)
	if err != nil {
		return models.LinkStats{}, fmt.Errorf("db query daily clicks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dc models.DailyClicks
		var day time.Time
		if err := rows.Scan(&day, &dc.Clicks); err != nil {
			return models.LinkStats{}, fmt.Errorf("db scan daily clicks: %w", err)
		}
		dc.Date = day.Format("2006-01-02")
		stats.ClicksPerDay = append(stats.ClicksPerDay, dc)
	}

	rows, err = r.db.Query(context.Background(),
		`SELECT referer, COUNT(*) AS cnt
		 FROM link_clicks
		 WHERE link_id = $1 AND referer != ''
		 GROUP BY referer ORDER BY cnt DESC LIMIT 5`,
		linkID,
	)
	if err != nil {
		return models.LinkStats{}, fmt.Errorf("db query referers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rc models.RefererCount
		if err := rows.Scan(&rc.Referer, &rc.Clicks); err != nil {
			return models.LinkStats{}, fmt.Errorf("db scan referers: %w", err)
		}
		stats.TopReferers = append(stats.TopReferers, rc)
	}

	return stats, nil
}