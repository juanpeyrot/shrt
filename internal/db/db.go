package db

import (
	"context"
	"fmt"
	"shrt/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(cfg config.DBConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	return pgxpool.New(context.Background(), dsn)
}
