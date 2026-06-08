package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"shrt/internal/config"
	"shrt/internal/db"
	"shrt/internal/handlers"
	"shrt/internal/repositories/memory"
	"shrt/internal/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" || appEnv == "development" {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: could not load .env file:", err)
		}
	}

	cfg := config.Load()

	dbpool, err := db.New(cfg.DB)
	if err != nil {
		return err
	}
	defer dbpool.Close()

	r := chi.NewRouter()

	healthHandler := handlers.NewHealthHandler(dbpool)
	r.Get("/health", healthHandler.Health)

	repo := memory.NewLinkRepository()
	svc := services.NewLinkService(repo)
	linkHandler := handlers.NewLinkHandler(svc)
	linkHandler.RegisterRoutes(r)

	log.Println("Server running on port:", cfg.ServerPort)
	return http.ListenAndServe(":"+cfg.ServerPort, r)
}
