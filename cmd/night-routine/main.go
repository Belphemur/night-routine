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
	appSignals "github.com/belphemur/night-routine/internal/signals"
	"github.com/belphemur/night-routine/internal/token"
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

	// Initialize token store
	tokenStore, err := database.NewTokenStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	// Initialize token manager
	tokenManager := token.NewTokenManager(tokenStore, cfg.OAuth)

	// Create scheduler
	sched := scheduler.New(cfg, tracker)

	// Initialize calendar manager
	calendarManager := calendar.NewManager(tokenStore, tokenManager, cfg.OAuth)

	// Initialize calendar service without requiring a token
	calSvc := calendar.New(cfg, tokenStore, sched, tokenManager)
	log.Printf("Calendar service initialized. Waiting for authentication...")

	// Initialize OAuth handler
	oauthHandler, err := handlers.NewOAuthHandler(cfg, tokenStore, tokenManager)
	if err != nil {
		return fmt.Errorf("failed to initialize OAuth handler: %w", err)
	}
	oauthHandler.RegisterRoutes()

	// Initialize base handler
	baseHandler, err := handlers.NewBaseHandler(cfg, tokenStore, tracker)
	if err != nil {
		return fmt.Errorf("failed to initialize base handler: %w", err)
	}

	// Initialize home handler
	homeHandler := handlers.NewHomeHandler(baseHandler)
	homeHandler.RegisterRoutes()

	// Initialize calendar handler with the calendar manager
	calendarHandler := handlers.NewCalendarHandler(baseHandler, cfg, calendarManager)
	calendarHandler.RegisterRoutes()

	// Initialize sync handler with calendar service
	syncHandler := handlers.NewSyncHandler(baseHandler, sched, tokenManager, calSvc)
	syncHandler.RegisterRoutes()

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

	// Set up webhook handler using the calendar service (will be initialized later)
	webhookHandler := &handlers.WebhookHandler{
		BaseHandler:     baseHandler,
		CalendarService: calSvc,
		Scheduler:       sched,
		Config:          cfg,
	}
	webhookHandler.RegisterRoutes()

	// Register handler for token setup signals
	appSignals.OnTokenSetup(func(ctx context.Context, data appSignals.TokenSetupData) {
		if data.Success {
			log.Printf("Token setup detected - initializing calendar service")

			// Initialize the calendar service with the new token
			if err := calSvc.Initialize(ctx); err != nil {
				log.Printf("Failed to initialize calendar service: %v", err)
				return
			}

			log.Printf("Calendar service initialized successfully")

			// We don't set up notification channels here anymore,
			// they will be set up when a calendar is selected
		}
	}, "main-token-setup-handler")

	// Register handler for calendar selection signals
	appSignals.OnCalendarSelected(func(ctx context.Context, data appSignals.CalendarSelectedData) {
		log.Printf("Calendar selection detected - setting up notification channel for calendar ID: %s", data.CalendarID)

		// Initialize calendar service if not already initialized
		if !calSvc.IsInitialized() {
			if err := calSvc.Initialize(ctx); err != nil {
				log.Printf("Failed to initialize calendar service on calendar selection: %v", err)
				return
			}
		}

		// Set up notification channel for calendar changes
		if err := calSvc.SetupNotificationChannel(ctx); err != nil {
			log.Printf("Warning: Failed to set up notification channel: %v", err)
		} else {
			log.Printf("Successfully set up notification channel for calendar changes")
		}

		// Update schedule immediately after calendar selection
		if err := updateSchedule(ctx, cfg, sched, calSvc); err != nil {
			log.Printf("Failed to update schedule after calendar selection: %v", err)
		}
	}, "main-calendar-selected-handler")

	// Main service loop
	ticker := time.NewTicker(getUpdateInterval(cfg.Schedule.UpdateFrequency))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Stop notification channels if calendar service is available
			if calSvc.IsInitialized() {
				if err := calSvc.StopAllNotificationChannels(ctx); err != nil {
					log.Printf("Warning: Failed to stop notification channels: %v", err)
				}
			}

			// Shutdown HTTP server
			if err := srv.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown error: %v", err)
			}
			return nil

		case <-ticker.C:
			if calSvc.IsInitialized() {
				if err := updateSchedule(ctx, cfg, sched, calSvc); err != nil {
					log.Printf("Failed to update schedule: %v", err)
				}
			} else {
				// Try to initialize calendar service if it wasn't available before
				if err := calSvc.Initialize(ctx); err != nil {
					log.Printf("Calendar service not ready: %v", err)
				} else {
					log.Printf("Calendar service initialized successfully on scheduled check")

					// We don't automatically set up notification channels here either,
					// as we want to wait for calendar selection
				}
			}
		}
	}
}

func updateSchedule(ctx context.Context, cfg *config.Config, sched *scheduler.Scheduler, calSvc *calendar.Service) error {
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
