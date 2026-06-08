package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"shrt/internal/handlers"
	"shrt/internal/repositories/memory"
	"shrt/internal/services"
)

func main() {
	appEnv := os.Getenv("APP_ENV")

	if appEnv == "" || appEnv == "development" {
		if err := godotenv.Load(); err != nil {
			fmt.Println("Error loading .env file:", err)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	r := chi.NewRouter()

	r.Get("/health", handlers.Health)

	repo := memory.NewLinkRepository()
	svc := services.NewLinkService(repo)
	linkHandler := handlers.NewLinkHandler(svc)
	linkHandler.RegisterRoutes(r)

	fmt.Println("Server running on: " + port)
	http.ListenAndServe(":"+port, r)
}