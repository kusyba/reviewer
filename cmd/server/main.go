package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"pr-reviewer-service/internal/handlers"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", getDBConnectionString())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	repo := repository.NewRepository(db)
	svc := service.NewService(repo)
	h := handlers.NewHandler(svc)

	router := h.SetupRoutes()

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(server.ListenAndServe())
}

func getDBConnectionString() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}
