package signals

import (
	"context"

	"github.com/maniartech/signals"
)

// TokenSetupData contains data associated with token setup signal
type TokenSetupData struct {
	// You can add additional fields here if needed
	Success bool
}

// CalendarSelectedData contains data associated with calendar selection signal
type CalendarSelectedData struct {
	CalendarID string
}

// Signal definitions using generics
var TokenSetup = signals.New[TokenSetupData]()
var CalendarSelected = signals.New[CalendarSelectedData]()

// EmitTokenSetup emits a signal when a token is successfully set up
func EmitTokenSetup(ctx context.Context, success bool) {
	TokenSetup.Emit(ctx, TokenSetupData{
		Success: success,
	})
}

// EmitCalendarSelected emits a signal when a calendar is selected
func EmitCalendarSelected(ctx context.Context, calendarID string) {
	CalendarSelected.Emit(ctx, CalendarSelectedData{
		CalendarID: calendarID,
	})
}

// OnTokenSetup registers a handler for token setup events
func OnTokenSetup(handler func(ctx context.Context, data TokenSetupData), key ...string) {
	if len(key) > 0 {
		TokenSetup.AddListener(handler, key[0])
	} else {
		TokenSetup.AddListener(handler)
	}
}

// OnCalendarSelected registers a handler for calendar selection events
func OnCalendarSelected(handler func(ctx context.Context, data CalendarSelectedData), key ...string) {
	if len(key) > 0 {
		CalendarSelected.AddListener(handler, key[0])
	} else {
		CalendarSelected.AddListener(handler)
	}
}
