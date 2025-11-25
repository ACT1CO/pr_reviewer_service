// cmd/server/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/lib/pq"

	"reviewer_service/internal/handlers"
	"reviewer_service/internal/repository"
	"reviewer_service/internal/service"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	if err := waitForDB(db); err != nil {
		log.Fatal("Database is not ready:", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("Failed to create PostgreSQL driver:", err)
	}

	sourceDriver, err := iofs.New(os.DirFS("."), "migrations")
	if err != nil {
		log.Fatal("Failed to open migrations folder:", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		log.Fatal("Failed to create migrate instance:", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Migration failed:", err)
	}
	log.Println("Migrations applied successfully")

	teamRepo := repository.NewTeamRepository(db)
	userRepo := repository.NewUserRepository(db)
	prRepo := repository.NewPullRequestRepository(db)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)
	teamService := service.NewTeamService(teamRepo, userRepo, prRepo, prService)
	userService := service.NewUserService(userRepo, teamRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.HealthHandler)
	mux.HandleFunc("POST /team/add", handlers.AddTeamHandler(teamService))
	mux.HandleFunc("POST /pullRequest/create", handlers.CreatePullRequestHandler(prService))
	mux.HandleFunc("POST /pullRequest/merge", handlers.MergePullRequestHandler(prService))
	mux.HandleFunc("POST /pullRequest/reassign", handlers.ReassignReviewerHandler(prService))
	mux.HandleFunc("GET /users/getReview", handlers.GetReviewPRsHandler(prService))
	mux.HandleFunc("GET /stats/reviews", handlers.GetReviewStatsHandler(prService))
	mux.HandleFunc("POST /team/deactivateUsers", handlers.DeactivateUsersHandler(teamService))
	mux.HandleFunc("POST /users/setIsActive", handlers.SetIsActiveHandler(userService))
	mux.HandleFunc("GET /team/get", handlers.GetTeamHandler(teamService))

	handler := handlers.LoggingMiddleware(mux)
	server := &http.Server{Addr: ":8080", Handler: handler}

	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited gracefully")
}

func waitForDB(db *sql.DB) error {
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			return nil
		}
		log.Println("Waiting for DB...")
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("database not ready after 30 seconds")
}
