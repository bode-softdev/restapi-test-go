// Package main menjalankan REST API manajemen produk.
//
//	@title						REST API Product Management
//	@version					1.0
//	@description				REST API untuk manajemen produk dengan autentikasi JWT, cache, dan rate limit.
//	@description				Seluruh endpoint /api/products wajib memakai header Authorization: Bearer <authentication_token>.
//	@termsOfService				http://swagger.io/terms/
//	@contact.name				restapi-test
//	@license.name				MIT
//	@host						localhost:8080
//	@BasePath					/
//	@schemes					http https
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Masukkan "Bearer {token}"
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"restapi-test/internal/config"
	"restapi-test/internal/database"
	"restapi-test/internal/handler"
	"restapi-test/internal/middleware"
	"restapi-test/internal/repository"
	"restapi-test/internal/router"
	"restapi-test/internal/service"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	gin.SetMode(cfg.GinMode)

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: koneksi gagal: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("database: tutup koneksi gagal: %v", err)
		}
	}()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("database: migrasi gagal: %v", err)
	}
	if err := database.Seed(db); err != nil {
		log.Printf("database: seeding dilewati: %v", err)
	}

	productRepo := repository.NewProductRepository(db)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)

	authService := service.NewAuthService(userRepo, tokenRepo, cfg)
	productService := service.NewProductService(productRepo)

	engine := router.New(router.Deps{
		Products:       handler.NewProductHandler(productService),
		Auth:           handler.NewAuthHandler(authService),
		AuthMiddleware: middleware.Auth(authService),
	})

	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("server: listen di %s (mode %s)", cfg.Addr(), cfg.GinMode)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: gagal listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("server: shutdown diminta")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server: shutdown paksa: %v", err)
	}
	log.Println("server: berhenti")
}
