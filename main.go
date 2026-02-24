package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"rental-architecture-back/internal/app"
	"rental-architecture-back/internal/controller"
)

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func main() {
	if err := godotenv.Load("dev.env"); err != nil {
		log.Printf("godotenv dev.env load: %v", err)
	}
	log.Printf("DB_PATH env (raw): %q", os.Getenv("DB_PATH"))

	addr := getenv("ADDR", ":8080")
	dbPath := getenv("DB_PATH", "./app.db")
	jwtSecret := getenv("JWT_SECRET", "dev-secret-change-me")
	tokenTTL := 24 * time.Hour

	a, err := app.New(app.Config{
		DBPath:    dbPath,
		JWTSecret: []byte(jwtSecret),
		TokenTTL:  tokenTTL,
	})
	if err != nil {
		log.Fatal(err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	ctrl := controller.New(controller.Deps{
		DB:        a.DB,
		Auth:      a.Auth,
		Audit:     a.Audit,
		Analytics: a.Analytics,
		Sim:       a.Sim,
	})
	ctrl.Mount(r)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
