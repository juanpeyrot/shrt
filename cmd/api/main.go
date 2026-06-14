package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"shrt/internal/config"
	"shrt/internal/db"
	"shrt/internal/handlers"
	"shrt/internal/middleware"
	"shrt/internal/repositories"
	"shrt/internal/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if getEnv("APP_ENV", "development") == "development" {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: could not load .env file:", err)
		}
	}

	cfg := config.New(
		config.WithServerPort(getEnv("PORT", "3000")),
		config.WithEnvironment(config.Environment(getEnv("APP_ENV", "development"))),
		config.WithMaxConn(parseUint(getEnv("MAX_CONN", "5"))),
		config.WithTLS(getEnv("TLS_ENABLED", "false") == "true"),
		config.WithDB(config.DBConfig{
			Host:     mustGetEnv("DB_HOST"),
			Port:     mustGetEnv("DB_PORT"),
			User:     mustGetEnv("DB_USER"),
			Password: mustGetEnv("DB_PASSWORD"),
			Name:     mustGetEnv("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		}),
	)

	dbpool, err := db.New(cfg.DB())
	if err != nil {
		return err
	}
	defer dbpool.Close()

	r := chi.NewRouter()

	healthHandler := handlers.NewHealthHandler(dbpool)
	r.Get("/health", healthHandler.Health)

	jwtSecret := []byte(mustGetEnv("JWT_SECRET"))

	authRepo := repositories.NewAuthRepository(dbpool)
	tokenSvc := services.NewTokenService(jwtSecret, authRepo)
	authSvc := services.NewAuthService(authRepo, tokenSvc)
	authHandler := handlers.NewAuthHandler(authSvc)
	r.Post("/sign-up", authHandler.RegisterLocal)

	linkRepo := repositories.NewLinkRepository(dbpool)
	linkSvc := services.NewLinkService(linkRepo)
	linkHandler := handlers.NewLinkHandler(linkSvc)

	r.Get("/{shortCode}", linkHandler.Redirect)
	r.Post("/links", linkHandler.CreateShortURL)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(jwtSecret))
	})

	log.Println("Server running on port:", cfg.ServerPort())
	return http.ListenAndServe(":"+cfg.ServerPort(), r)
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseUint(s string) uint {
	n, _ := strconv.ParseUint(s, 10, 64)
	return uint(n)
}
