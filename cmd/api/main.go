package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"shrt/internal/auth/oauth"
	"shrt/internal/cache"
	"shrt/internal/config"
	"shrt/internal/db"
	"shrt/internal/handlers"
	"shrt/internal/middleware"
	"shrt/internal/repositories"
	"shrt/internal/services"
	"shrt/internal/web"
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
		config.WithBaseURL(getEnv("BASE_URL", "http://localhost:3000")),
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
		config.WithOAuth(config.OAuthConfig{
			Google: config.ProviderConfig{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
			},
			Github: config.ProviderConfig{
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITHUB_REDIRECT_URL", ""),
			},
		}),
		config.WithRedis(config.RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       int(parseUint(getEnv("REDIS_DB", "0"))),
		}),
	)

	dbpool, err := db.New(cfg.DB())
	if err != nil {
		return err
	}
	defer dbpool.Close()

	redisClient, err := cache.New(cfg.Redis())
	if err != nil {
		return err
	}
	defer redisClient.Close()

	linkCache := cache.NewLinkCache(redisClient)

	r := chi.NewRouter()

	// Health
	healthHandler := handlers.NewHealthHandler(dbpool)
	r.Get("/health", healthHandler.Health)

	jwtSecret := []byte(mustGetEnv("JWT_SECRET"))

	// Repositories & services
	authRepo := repositories.NewAuthRepository(dbpool)
	tokenSvc := services.NewTokenService(jwtSecret, authRepo)
	authSvc := services.NewAuthService(authRepo, tokenSvc)
	oauthRegistry := buildOAuthRegistry(cfg.OAuth())

	linkRepo := repositories.NewLinkRepository(dbpool)
	linkSvc := services.NewLinkService(linkRepo, linkCache)

	// --- API routes ---
	authHandler := handlers.NewAuthHandler(authSvc, oauthRegistry)
	linkHandler := handlers.NewLinkHandler(linkSvc)

	r.Route("/api", func(r chi.Router) {
		r.Post("/sign-up", authHandler.RegisterLocal)
		r.Post("/login", authHandler.LoginLocal)
		r.Get("/auth/{provider}", authHandler.OAuthRedirect)
		r.Get("/auth/{provider}/callback", authHandler.OAuthCallback)

		r.With(middleware.OptionalAuthenticate(jwtSecret)).Post("/links", linkHandler.CreateShortURL)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(jwtSecret))
			r.Get("/links", linkHandler.ListLinks)
			r.Get("/links/{shortCode}", linkHandler.RetrieveOriginalURL)
			r.Put("/links/{shortCode}", linkHandler.UpdateShortURL)
			r.Delete("/links/{shortCode}", linkHandler.DeleteShortURL)
			r.Get("/links/{shortCode}/stats", linkHandler.GetStats)
		})
	})

	// --- Web UI ---
	tmpl, err := web.NewTemplateRegistry("ui/templates", cfg.BaseURL())
	if err != nil {
		return err
	}

	webHandler := web.NewWebHandler(linkSvc, authSvc, tokenSvc, oauthRegistry, tmpl, jwtSecret, cfg.BaseURL(), cfg.TLS())

	// Static files
	fileServer := http.FileServer(http.Dir("ui/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public web pages
	r.Group(func(r chi.Router) {
		r.Use(web.WebOptionalAuth(jwtSecret, tokenSvc, authSvc, cfg.TLS()))
		r.Get("/", webHandler.Home)
		r.Post("/shorten", webHandler.CreateShortURL)
	})

	// Auth web pages
	r.Get("/login", webHandler.LoginPage)
	r.Post("/login", webHandler.LoginSubmit)
	r.Get("/register", webHandler.RegisterPage)
	r.Post("/register", webHandler.RegisterSubmit)
	r.Get("/logout", webHandler.Logout)
	r.Get("/auth/{provider}/start", webHandler.WebOAuthStart)

	// Protected web pages
	r.Group(func(r chi.Router) {
		r.Use(web.WebAuthenticate(jwtSecret, tokenSvc, authSvc, cfg.TLS()))
		r.Get("/dashboard", webHandler.Dashboard)
		r.Get("/dashboard/links", webHandler.LinkTable)
		r.Delete("/dashboard/links/{shortCode}", webHandler.DeleteLink)
		r.Get("/dashboard/links/{shortCode}/edit", webHandler.EditForm)
		r.Put("/dashboard/links/{shortCode}", webHandler.UpdateLink)
		r.Get("/dashboard/links/{shortCode}/row", webHandler.LinkRow)
		r.Get("/dashboard/links/{shortCode}/stats", webHandler.LinkStats)
	})

	// Redirect catch-all
	r.Get("/{shortCode}", linkHandler.Redirect)

	log.Println("Server running on port:", cfg.ServerPort())
	return http.ListenAndServe(":"+cfg.ServerPort(), r)
}

func buildOAuthRegistry(cfg config.OAuthConfig) oauth.Registry {
	registry := oauth.Registry{}
	if cfg.Google.Enabled() {
		registry["google"] = oauth.NewGoogleProvider(
			cfg.Google.ClientID, cfg.Google.ClientSecret, cfg.Google.RedirectURL,
		)
	}
	if cfg.Github.Enabled() {
		registry["github"] = oauth.NewGithubProvider(
			cfg.Github.ClientID, cfg.Github.ClientSecret, cfg.Github.RedirectURL,
		)
	}
	return registry
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
