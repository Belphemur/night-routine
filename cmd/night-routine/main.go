package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/handlers"
	"github.com/belphemur/night-routine/internal/scheduler"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {

	log.Printf("Starting Night Routine Scheduler v%s (%s) built at %s", version, commit, date)

	// Create context that's canceled on SIGINT/SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, initiating shutdown", sig)
		cancel()
	}()

	if err := run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(ctx context.Context) error {
	// Get config file path from environment or use default
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "configs/routine.toml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(cfg.Service.StateFile), 0755); err != nil {
		return err
	}

	// Initialize database
	db, err := database.New(cfg.Service.StateFile)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitSchema(); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	// Initialize fairness tracker
	tracker, err := fairness.New(db)
	if err != nil {
		return err
	}
	defer tracker.Close()

	// Initialize token store
	tokenStore, err := database.NewTokenStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	// Initialize OAuth handler
	oauthHandler, err := handlers.NewOAuthHandler(cfg, tokenStore)
	if err != nil {
		return fmt.Errorf("failed to initialize OAuth handler: %w", err)
	}
	oauthHandler.RegisterRoutes()

	// Create scheduler
	sched := scheduler.New(cfg, tracker)

	// Start HTTP server
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.App.Port),
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting OAuth web server on port %d", cfg.App.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Initialize calendar service with token store
	calSvc, err := calendar.New(ctx, cfg, tokenStore)
	if err != nil {
		if err.Error() == "no token available - please authenticate via web interface first" {
			log.Printf("Please visit http://localhost:%d to authenticate with Google Calendar", cfg.App.Port)
		} else {
			return fmt.Errorf("failed to create calendar service: %w", err)
		}
	}

	// Main service loop
	ticker := time.NewTicker(getUpdateInterval(cfg.Schedule.UpdateFrequency))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Shutdown HTTP server
			if err := srv.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown error: %v", err)
			}
			return nil

		case <-ticker.C:
			if calSvc != nil {
				if err := updateSchedule(ctx, cfg, sched, calSvc, tracker); err != nil {
					log.Printf("Failed to update schedule: %v", err)
				}
			} else {
				// Try to initialize calendar service if it wasn't available before
				calSvc, err = calendar.New(ctx, cfg, tokenStore)
				if err != nil {
					log.Printf("Calendar service not ready: %v", err)
				}
			}
		}
	}
}

func updateSchedule(ctx context.Context, cfg *config.Config, sched *scheduler.Scheduler, calSvc *calendar.Service, tracker *fairness.Tracker) error {
	// Calculate date range
	now := time.Now()
	end := now.AddDate(0, 0, cfg.Schedule.LookAheadDays)

	// Generate schedule
	assignments, err := sched.GenerateSchedule(now, end)
	if err != nil {
		return err
	}

	// Sync with calendar
	if err := calSvc.SyncSchedule(ctx, assignments); err != nil {
		return err
	}

	// Record assignments
	for _, a := range assignments {
		if err := tracker.RecordAssignment(a.Parent, a.Date); err != nil {
			return err
		}
	}

	log.Printf("Updated schedule for %d days with %d assignments", cfg.Schedule.LookAheadDays, len(assignments))
	return nil
}

func getUpdateInterval(frequency string) time.Duration {
	switch frequency {
	case "daily":
		return 24 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	case "monthly":
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}
