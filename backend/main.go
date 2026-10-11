package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/syahnur197/my-gym-app/backend/internal/auth"
	"github.com/syahnur197/my-gym-app/backend/internal/database"
	"github.com/syahnur197/my-gym-app/backend/internal/repository"
	"github.com/syahnur197/my-gym-app/backend/internal/service"
	"github.com/syahnur197/my-gym-app/backend/internal/web"
)

func main() {
	loadEnv(".env")
	username, password, secret := os.Getenv("APP_USERNAME"), os.Getenv("APP_PASSWORD"), os.Getenv("JWT_SECRET")
	if username == "" || len(password) < 12 || len(secret) < 32 {
		log.Fatal("configure APP_USERNAME, APP_PASSWORD (12+ characters), JWT_SECRET (32+ characters)")
	}
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "./gym.db"
	}
	db, err := database.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	svc := service.New(repository.NewWorkout(db), repository.NewSession(db))
	if err := svc.Seed(context.Background()); err != nil {
		log.Fatal(err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	authManager := auth.New(username, password, secret)
	webMux := http.NewServeMux()
	web.New(svc, authManager, os.Getenv("COOKIE_SECURE") == "true").Register(webMux)

	server := &http.Server{Addr: ":" + port, Handler: webMux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("My Gym listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}
func loadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}
