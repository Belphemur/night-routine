package handlers

// Error Codes
const (
	ErrCodeInvalidFormData           = "invalid_form_data"
	ErrCodeInvalidDayOfWeek          = "invalid_day_of_week"
	ErrCodeInvalidLookAheadDays      = "invalid_look_ahead_days"
	ErrCodeInvalidPastEventThreshold = "invalid_past_event_threshold"
	ErrCodeFailedSaveParent          = "failed_save_parent"
	ErrCodeFailedSaveAvailability    = "failed_save_availability"
	ErrCodeFailedSaveSchedule        = "failed_save_schedule"
	ErrCodeSyncFailed                = "sync_failed"
	ErrCodeAuthRequired              = "authentication_required"
	ErrCodeCalendarSelectionRequired = "calendar_selection_required"
	ErrCodeCalendarClientError       = "calendar_client_error"
	ErrCodeCalendarFetchError        = "calendar_fetch_error"
	ErrCodeCalendarGenerationError   = "calendar_generation_error"
	ErrCodeUnknown                   = "unknown_error"
	ErrCodeUnauthorized              = "unauthorized"
	ErrCodeMissingAssignmentID       = "missing_assignment_id"
	ErrCodeInvalidAssignmentID       = "invalid_assignment_id"
	ErrCodeUnlockFailed              = "unlock_failed"
	ErrCodeNotOverridden             = "not_overridden"
)

// Success Codes
const (
	SuccessCodeSettingsUpdated           = "settings_updated"
	SuccessCodeSettingsUpdatedSyncFailed = "settings_updated_sync_failed"
	SuccessCodeSyncComplete              = "sync_complete"
	SuccessCodeAssignmentUnlocked        = "assignment_unlocked"
)

// ErrorMessages maps error codes to user-friendly messages
var ErrorMessages = map[string]string{
	ErrCodeInvalidFormData:           "Invalid form data.",
	ErrCodeInvalidDayOfWeek:          "Invalid day of week.",
	ErrCodeInvalidLookAheadDays:      "Look ahead days must be between 1 and 365.",
	ErrCodeInvalidPastEventThreshold: "Past event threshold must be between 0 and 30.",
	ErrCodeFailedSaveParent:          "Failed to save parent names.",
	ErrCodeFailedSaveAvailability:    "Failed to save availability.",
	ErrCodeFailedSaveSchedule:        "Failed to save schedule settings.",
	ErrCodeSyncFailed:                "Failed to sync schedule. Please try again.",
	ErrCodeAuthRequired:              "Authentication required. Please connect your Google Calendar first.",
	ErrCodeCalendarSelectionRequired: "Please select a calendar first.",
	ErrCodeCalendarClientError:       "Failed to connect to Google Calendar. Please try authenticating again.",
	ErrCodeCalendarFetchError:        "Failed to fetch your calendars. Please try authenticating again.",
	ErrCodeCalendarGenerationError:   "Failed to generate the assignment calendar. Please check logs or try again later.",
	ErrCodeUnknown:                   "An unknown error occurred.",
	ErrCodeUnauthorized:              "You must be logged in to perform this action.",
	ErrCodeMissingAssignmentID:       "No assignment specified.",
	ErrCodeInvalidAssignmentID:       "Invalid assignment ID.",
	ErrCodeUnlockFailed:              "Failed to unlock assignment. Please try again.",
	ErrCodeNotOverridden:             "Cannot unlock an assignment that hasn't been manually overridden.",
}

// SuccessMessages maps success codes to user-friendly messages
var SuccessMessages = map[string]string{
	SuccessCodeSettingsUpdated:           "Settings updated and schedule synced successfully.",
	SuccessCodeSettingsUpdatedSyncFailed: "Settings updated but sync failed. Please sync manually.",
	SuccessCodeSyncComplete:              "Schedule successfully synced with Google Calendar.",
	SuccessCodeAssignmentUnlocked:        "Assignment unlocked successfully.",
}

// GetErrorMessage returns the message for a given error code
func GetErrorMessage(code string) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return ErrorMessages[ErrCodeUnknown]
}

// GetSuccessMessage returns the message for a given success code
func GetSuccessMessage(code string) string {
	if msg, ok := SuccessMessages[code]; ok {
		return msg
	}
	return ""
}
