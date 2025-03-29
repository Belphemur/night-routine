package main

import (
	"context"
	"fmt"
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
	"github.com/belphemur/night-routine/internal/logging" // Import the new logging package
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
	// Determine if we're in development mode
	isDev := os.Getenv("ENV") != "production"

	// Initialize logging
	logging.Initialize(isDev)

	// Get a logger for the main component
	logger := logging.GetLogger("main")

	logger.Info().
		Str("version", version).
		Str("commit", commit).
		Str("build_date", date).
		Msg("Starting Night Routine Scheduler")

	// Create context that's canceled on SIGINT/SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info().Str("signal", sig.String()).Msg("Received signal, initiating shutdown")
		cancel()
	}()

	if err := run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Application run failed")
	}
}

func run(ctx context.Context) error {
	// Get logger for the run function
	logger := logging.GetLogger("main")

	// Get config file path from environment or use default
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "configs/routine.toml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		// Log error before returning, as main's fatal log won't have config context
		logger.Error().Err(err).Str("config_path", configPath).Msg("Failed to load configuration")
		return err
	}

	// Set log level from configuration
	logging.SetLogLevel(cfg.Service.LogLevel)
	logger.Info().Str("log_level", cfg.Service.LogLevel).Msg("Log level set")

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(cfg.Service.StateFile), 0755); err != nil {
		logger.Error().Err(err).Str("path", filepath.Dir(cfg.Service.StateFile)).Msg("Failed to create data directory")
		return err
	}

	// Initialize database
	db, err := database.New(cfg.Service.StateFile)
	if err != nil {
		// Wrap error for context, logger will handle Err field
		wrappedErr := fmt.Errorf("failed to initialize database: %w", err)
		logger.Error().Err(wrappedErr).Str("db_path", cfg.Service.StateFile).Msg("Database initialization failed")
		return wrappedErr
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitSchema(); err != nil {
		wrappedErr := fmt.Errorf("failed to initialize database schema: %w", err)
		logger.Error().Err(wrappedErr).Msg("Database schema initialization failed")
		return wrappedErr
	}

	// Initialize fairness tracker
	tracker, err := fairness.New(db)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize fairness tracker")
		return err // Return original error
	}

	// Initialize token store
	tokenStore, err := database.NewTokenStore(db)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to initialize token store: %w", err)
		logger.Error().Err(wrappedErr).Msg("Token store initialization failed")
		return wrappedErr
	}

	// Initialize token manager
	tokenManager := token.NewTokenManager(tokenStore, cfg.OAuth)

	// Create scheduler
	sched := scheduler.New(cfg, tracker)

	// Initialize calendar manager
	calendarManager := calendar.NewManager(tokenStore, tokenManager, cfg.OAuth)

	// Initialize calendar service without requiring a token
	calSvc := calendar.New(cfg, tokenStore, sched, tokenManager)
	logger.Info().Msg("Calendar service created. Waiting for authentication/initialization...")

	// Initialize OAuth handler
	oauthHandler, err := handlers.NewOAuthHandler(cfg, tokenStore, tokenManager)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to initialize OAuth handler: %w", err)
		logger.Error().Err(wrappedErr).Msg("OAuth handler initialization failed")
		return wrappedErr
	}
	oauthHandler.RegisterRoutes()

	// Initialize base handler
	baseHandler, err := handlers.NewBaseHandler(cfg, tokenStore, tokenManager, tracker)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to initialize base handler: %w", err)
		logger.Error().Err(wrappedErr).Msg("Base handler initialization failed")
		return wrappedErr
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
		logger.Info().Int("port", cfg.App.Port).Msg("Starting OAuth web server")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Set up webhook handler using the calendar service (will be initialized later)
	webhookHandler := handlers.NewWebhookHandler(baseHandler, calSvc, sched, cfg, tokenManager)
	webhookHandler.RegisterRoutes()

	hasToken, _ := tokenManager.HasToken()
	if hasToken {
		logger.Info().Msg("Token found, attempting initial calendar service initialization and notification setup")
		// Initialize calendar service if not already initialized
		if !calSvc.IsInitialized() {
			if err := calSvc.Initialize(ctx); err != nil {
				// Log as warning, app can continue without calendar initially
				logger.Warn().Err(err).Msg("Initial calendar service initialization failed")
				// Do not return err here, allow app to run
			} else {
				logger.Info().Msg("Initial calendar service initialization successful")
				// Set up notification channel for calendar changes only if initialized
				if err := calSvc.SetupNotificationChannel(ctx); err != nil {
					logger.Warn().Err(err).Msg("Failed to set up notification channel after initial check")
				} else {
					logger.Info().Msg("Successfully set up notification channel after initial check")
				}
			}
		} else {
			logger.Info().Msg("Calendar service already initialized")
			// Ensure notification channel is set up if already initialized
			if err := calSvc.SetupNotificationChannel(ctx); err != nil {
				logger.Warn().Err(err).Msg("Failed to set up notification channel (service already initialized)")
			} else {
				logger.Info().Msg("Successfully set up notification channel (service already initialized)")
			}
		}
	} else {
		logger.Info().Msg("No token found initially. Waiting for OAuth flow.")
	}

	// Register handler for token setup signals
	appSignals.OnTokenSetup(func(ctx context.Context, data appSignals.TokenSetupData) {
		signalLogger := logging.GetLogger("signal-token-setup")
		if data.Success {
			signalLogger.Info().Msg("Token setup detected - initializing calendar service")

			// Initialize the calendar service with the new token
			if err := calSvc.Initialize(ctx); err != nil {
				signalLogger.Error().Err(err).Msg("Failed to initialize calendar service after token setup")
				return
			}

			signalLogger.Info().Msg("Calendar service initialized successfully after token setup")

			// We don't set up notification channels here anymore,
			// they will be set up when a calendar is selected
		} else {
			signalLogger.Warn().Msg("Token setup signal received, but setup was not successful")
		}
	}, "main-token-setup-handler")

	// Register handler for calendar selection signals
	appSignals.OnCalendarSelected(func(ctx context.Context, data appSignals.CalendarSelectedData) {
		signalLogger := logging.GetLogger("signal-calendar-selected")
		signalLogger.Info().Str("calendar_id", data.CalendarID).Msg("Calendar selection detected - setting up notification channel")

		// Initialize calendar service if not already initialized (should be rare here)
		if !calSvc.IsInitialized() {
			signalLogger.Warn().Msg("Calendar service not initialized during calendar selection, attempting initialization")
			if err := calSvc.Initialize(ctx); err != nil {
				signalLogger.Error().Err(err).Msg("Failed to initialize calendar service on calendar selection")
				return
			}
			signalLogger.Info().Msg("Calendar service initialized successfully during calendar selection")
		}

		// Set up notification channel for calendar changes
		if err := calSvc.SetupNotificationChannel(ctx); err != nil {
			signalLogger.Warn().Err(err).Msg("Failed to set up notification channel after calendar selection")
		} else {
			signalLogger.Info().Msg("Successfully set up notification channel after calendar selection")
		}

		// Update schedule immediately after calendar selection
		if err := updateSchedule(ctx, cfg, sched, calSvc); err != nil {
			signalLogger.Error().Err(err).Msg("Failed to update schedule after calendar selection")
		}
	}, "main-calendar-selected-handler")

	// Main service loop
	ticker := time.NewTicker(getUpdateInterval(cfg.Schedule.UpdateFrequency))
	defer ticker.Stop()

	logger.Info().Str("update_frequency", cfg.Schedule.UpdateFrequency).Msg("Starting main service loop")
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Context cancelled, initiating shutdown sequence")
			// Stop notification channels if calendar service is available
			if calSvc.IsInitialized() {
				logger.Info().Msg("Stopping notification channels...")
				if err := calSvc.StopAllNotificationChannels(context.Background()); err != nil { // Use background context for shutdown
					logger.Warn().Err(err).Msg("Failed to stop notification channels")
				} else {
					logger.Info().Msg("Notification channels stopped")
				}
			}

			// Shutdown HTTP server
			logger.Info().Msg("Shutting down HTTP server...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				logger.Error().Err(err).Msg("HTTP server shutdown error")
			} else {
				logger.Info().Msg("HTTP server shut down gracefully")
			}
			logger.Info().Msg("Shutdown complete")
			return nil

		case <-ticker.C:
			logger.Debug().Msg("Update schedule tick received")
			if calSvc.IsInitialized() {
				if err := updateSchedule(ctx, cfg, sched, calSvc); err != nil {
					logger.Error().Err(err).Msg("Failed to update schedule on tick")
				}
			} else {
				logger.Debug().Msg("Calendar service not initialized, attempting initialization on tick")
				// Try to initialize calendar service if it wasn't available before
				if err := calSvc.Initialize(ctx); err != nil {
					logger.Warn().Err(err).Msg("Calendar service still not ready")
				} else {
					logger.Info().Msg("Calendar service initialized successfully on scheduled check")
					// Notification channel setup will happen on calendar selection
				}
			}
		}
	}
}

func updateSchedule(ctx context.Context, cfg *config.Config, sched *scheduler.Scheduler, calSvc *calendar.Service) error {
	scheduleLogger := logging.GetLogger("schedule-update")
	scheduleLogger.Info().Msg("Starting schedule update")
	// Calculate date range
	now := time.Now()
	end := now.AddDate(0, 0, cfg.Schedule.LookAheadDays)
	scheduleLogger.Debug().Time("start_date", now).Time("end_date", end).Int("lookahead_days", cfg.Schedule.LookAheadDays).Msg("Calculated date range")

	// Generate schedule
	assignments, err := sched.GenerateSchedule(now, end)
	if err != nil {
		scheduleLogger.Error().Err(err).Msg("Failed to generate schedule")
		return err
	}
	scheduleLogger.Info().Int("assignments_generated", len(assignments)).Msg("Schedule generated")

	// Sync with calendar
	if err := calSvc.SyncSchedule(ctx, assignments); err != nil {
		scheduleLogger.Error().Err(err).Msg("Failed to sync schedule with calendar")
		return err
	}

	scheduleLogger.Info().Int("days", cfg.Schedule.LookAheadDays).Int("assignments", len(assignments)).Msg("Updated schedule successfully")
	return nil
}

func getUpdateInterval(frequency string) time.Duration {
	switch frequency {
	case "daily":
		return 24 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	case "monthly":
		return 30 * 24 * time.Hour // Approximation
	default:
		logger := logging.GetLogger("main")
		logger.Warn().Str("frequency", frequency).Msg("Invalid update frequency specified, defaulting to daily")
		return 24 * time.Hour
	}
}
