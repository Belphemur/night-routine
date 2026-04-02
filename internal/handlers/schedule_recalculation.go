package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/rs/zerolog"
)

// recalculateScheduleAndSync regenerates assignments from fromDate and syncs
// assignments that already have Google Calendar event IDs.
func recalculateScheduleAndSync(
	ctx context.Context,
	logger zerolog.Logger,
	tracker fairness.TrackerInterface,
	scheduler Scheduler.SchedulerInterface,
	calendarService calendar.CalendarService,
	configStore config.ConfigStoreInterface,
	fromDate time.Time,
) error {
	recalcLogger := logger.With().Str("from_date", fromDate.Format("2006-01-02")).Logger()
	recalcLogger.Info().Msg("Recalculating schedule")

	recalcLogger.Debug().Msg("Fetching last assignment date from tracker")
	lastAssignmentDate, err := tracker.GetLastAssignmentDate()
	if err != nil {
		recalcLogger.Error().Err(err).Msg("Failed to get last assignment date")
		return fmt.Errorf("failed to get last assignment date: %w", err)
	}
	recalcLogger.Debug().Time("last_assignment_date", lastAssignmentDate).Msg("Retrieved last assignment date")

	endDate := lastAssignmentDate
	if endDate.IsZero() || endDate.Before(fromDate) {
		recalcLogger.Debug().Msg("Using look-ahead configuration to determine recalculation end date")
		_, lookAheadDays, _, _, err := configStore.GetSchedule()
		if err != nil {
			recalcLogger.Error().Err(err).Msg("Failed to get schedule configuration")
			return fmt.Errorf("failed to get schedule configuration: %w", err)
		}
		endDate = fromDate.AddDate(0, 0, lookAheadDays)
		recalcLogger.Debug().Int("look_ahead_days", lookAheadDays).Time("end_date", endDate).Msg("Calculated end date from look-ahead settings")
	} else {
		recalcLogger.Debug().Time("end_date", endDate).Msg("Using last assignment date as recalculation end date")
	}

	recalcLogger.Debug().Time("start_date", fromDate).Time("end_date", endDate).Msg("Generating schedule for recalculation window")
	assignments, err := scheduler.GenerateSchedule(fromDate, endDate, time.Now())
	if err != nil {
		recalcLogger.Error().Err(err).Msg("Failed to generate schedule during recalculation")
		return fmt.Errorf("failed to generate schedule: %w", err)
	}
	recalcLogger.Info().Int("assignments_generated", len(assignments)).Msg("Generated schedule during recalculation")

	var withEventIDs []*Scheduler.Assignment
	for _, a := range assignments {
		if a.GoogleCalendarEventID != "" {
			withEventIDs = append(withEventIDs, a)
		}
	}
	recalcLogger.Info().Int("assignments_with_event_ids", len(withEventIDs)).Msg("Filtered assignments with Google Calendar event IDs")

	recalcLogger.Debug().Msg("Syncing recalculated assignments with calendar")
	if err := calendarService.SyncSchedule(ctx, withEventIDs); err != nil {
		recalcLogger.Error().Err(err).Msg("Failed to sync recalculated assignments")
		return fmt.Errorf("failed to sync schedule: %w", err)
	}

	recalcLogger.Info().Msg("Schedule recalculation and sync completed")
	return nil
}
