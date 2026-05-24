package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	_ "modernc.org/sqlite"
)

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name       string
		assignment *scheduler.Assignment
		want       string
	}{
		{
			name: "parent assignment uses Parent field",
			assignment: &scheduler.Assignment{
				Parent:        "Alice",
				CaregiverType: fairness.CaregiverTypeParent,
			},
			want: "Alice",
		},
		{
			name: "babysitter assignment uses Parent name",
			assignment: &scheduler.Assignment{
				Parent:        "Dawn",
				CaregiverType: fairness.CaregiverTypeBabysitter,
			},
			want: "Dawn",
		},
		{
			name: "babysitter with parent name",
			assignment: &scheduler.Assignment{
				Parent:        "Dawn",
				CaregiverType: fairness.CaregiverTypeBabysitter,
			},
			want: "Dawn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, displayName(tt.assignment))
		})
	}
}

func TestFormatEventSummary(t *testing.T) {
	tests := []struct {
		name       string
		assignment *scheduler.Assignment
		want       string
	}{
		{
			name: "parent assignment",
			assignment: &scheduler.Assignment{
				Parent:        "Alice",
				CaregiverType: fairness.CaregiverTypeParent,
			},
			want: "[Alice] \U0001f303\U0001f476Routine",
		},
		{
			name: "babysitter assignment",
			assignment: &scheduler.Assignment{
				Parent:        "Dawn",
				CaregiverType: fairness.CaregiverTypeBabysitter,
			},
			want: "[Dawn] \U0001f303\U0001f476Routine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatEventSummary(tt.assignment))
		})
	}
}

func TestFormatEventDescription(t *testing.T) {
	tests := []struct {
		name       string
		assignment *scheduler.Assignment
		wantPrefix string
		wantSuffix string
	}{
		{
			name: "parent assignment says assigned to",
			assignment: &scheduler.Assignment{
				Parent:         "Alice",
				CaregiverType:  fairness.CaregiverTypeParent,
				DecisionReason: fairness.DecisionReasonTotalCount,
			},
			wantPrefix: "Night routine duty assigned to Alice",
			wantSuffix: "[" + constants.NightRoutineIdentifier + "]",
		},
		{
			name: "babysitter assignment says handled by babysitter",
			assignment: &scheduler.Assignment{
				Parent:         "Dawn",
				CaregiverType:  fairness.CaregiverTypeBabysitter,
				DecisionReason: fairness.DecisionReasonOverride,
			},
			wantPrefix: "Night routine handled by babysitter Dawn",
			wantSuffix: "[" + constants.NightRoutineIdentifier + "]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := formatEventDescription(tt.assignment)
			assert.Contains(t, desc, tt.wantPrefix)
			assert.Contains(t, desc, tt.wantSuffix)
		})
	}
}

type calendarTestConfigStore struct {
	parentA string
	parentB string
}

func (s *calendarTestConfigStore) GetParents() (string, string, error) {
	return s.parentA, s.parentB, nil
}

func (s *calendarTestConfigStore) GetAvailability(parent string) ([]string, error) {
	return nil, nil
}

func (s *calendarTestConfigStore) GetSchedule() (string, int, int, constants.StatsOrder, error) {
	return "weekly", 7, 5, constants.StatsOrderDesc, nil
}

func (s *calendarTestConfigStore) GetOAuthConfig() *oauth2.Config {
	return nil
}

func setupCalendarTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	db, err := database.New(database.SQLiteOptions{
		Path:        ":memory:",
		Mode:        "memory",
		Cache:       database.CacheShared,
		ForeignKeys: true,
		Journal:     database.JournalMemory,
		BusyTimeout: 5000,
	})
	require.NoError(t, err)
	require.NoError(t, db.MigrateDatabase())

	return db, func() {
		require.NoError(t, db.Close())
	}
}

type fakeCalendarAPI struct {
	t      *testing.T
	mu     sync.Mutex
	events map[string]*gcalendar.Event
	nextID int
}

func newFakeCalendarAPI(t *testing.T, events ...*gcalendar.Event) *fakeCalendarAPI {
	t.Helper()

	api := &fakeCalendarAPI{
		t:      t,
		events: make(map[string]*gcalendar.Event, len(events)),
		nextID: 1,
	}
	for _, event := range events {
		cloned := cloneEvent(t, event)
		api.events[cloned.Id] = cloned
	}
	return api
}

func (f *fakeCalendarAPI) handle(w http.ResponseWriter, r *http.Request) {
	f.t.Helper()

	idx := strings.Index(r.URL.Path, "/calendars/")
	if idx == -1 {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path[idx+len("/calendars/"):], "/"), "/")
	if len(parts) < 2 || parts[1] != "events" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if len(parts) == 2 {
			f.handleList(w)
			return
		}
		if len(parts) == 3 {
			f.handleGet(w, parts[2])
			return
		}
	case http.MethodPost:
		if len(parts) == 2 {
			f.handleInsert(w, r)
			return
		}
	case http.MethodPut:
		if len(parts) == 3 {
			f.handleUpdate(w, r, parts[2])
			return
		}
	case http.MethodDelete:
		if len(parts) == 3 {
			f.handleDelete(w, parts[2])
			return
		}
	}

	http.NotFound(w, r)
}

func (f *fakeCalendarAPI) handleList(w http.ResponseWriter) {
	f.mu.Lock()
	items := make([]*gcalendar.Event, 0, len(f.events))
	for _, event := range f.events {
		items = append(items, cloneEvent(f.t, event))
	}
	f.mu.Unlock()

	writeJSONResponse(f.t, w, http.StatusOK, &gcalendar.Events{Items: items})
}

func (f *fakeCalendarAPI) handleGet(w http.ResponseWriter, eventID string) {
	f.mu.Lock()
	event, ok := f.events[eventID]
	f.mu.Unlock()
	if !ok {
		http.Error(w, `{"error":{"code":404,"message":"not found"}}`, http.StatusNotFound)
		return
	}

	writeJSONResponse(f.t, w, http.StatusOK, cloneEvent(f.t, event))
}

func (f *fakeCalendarAPI) handleInsert(w http.ResponseWriter, r *http.Request) {
	var event gcalendar.Event
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&event))

	f.mu.Lock()
	event.Id = fmt.Sprintf("created-%d", f.nextID)
	f.nextID++
	stored := cloneEvent(f.t, &event)
	f.events[stored.Id] = stored
	f.mu.Unlock()

	writeJSONResponse(f.t, w, http.StatusOK, cloneEvent(f.t, stored))
}

func (f *fakeCalendarAPI) handleUpdate(w http.ResponseWriter, r *http.Request, eventID string) {
	var event gcalendar.Event
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&event))

	f.mu.Lock()
	if _, ok := f.events[eventID]; !ok {
		f.mu.Unlock()
		http.Error(w, `{"error":{"code":404,"message":"not found"}}`, http.StatusNotFound)
		return
	}
	event.Id = eventID
	stored := cloneEvent(f.t, &event)
	f.events[eventID] = stored
	f.mu.Unlock()

	writeJSONResponse(f.t, w, http.StatusOK, cloneEvent(f.t, stored))
}

func (f *fakeCalendarAPI) handleDelete(w http.ResponseWriter, eventID string) {
	f.mu.Lock()
	if _, ok := f.events[eventID]; !ok {
		f.mu.Unlock()
		http.Error(w, `{"error":{"code":404,"message":"not found"}}`, http.StatusNotFound)
		return
	}
	delete(f.events, eventID)
	f.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeCalendarAPI) event(t *testing.T, eventID string) *gcalendar.Event {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	event, ok := f.events[eventID]
	require.True(t, ok, "event %s should exist", eventID)
	return cloneEvent(t, event)
}

func (f *fakeCalendarAPI) eventCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, statusCode int, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func cloneEvent(t *testing.T, event *gcalendar.Event) *gcalendar.Event {
	t.Helper()

	if event == nil {
		return nil
	}

	raw, err := json.Marshal(event)
	require.NoError(t, err)

	var cloned gcalendar.Event
	require.NoError(t, json.Unmarshal(raw, &cloned))
	return &cloned
}

func newSyncTestService(t *testing.T, initialEvents ...*gcalendar.Event) (*Service, *fakeCalendarAPI, *scheduler.Scheduler, *fairness.Tracker, func()) {
	t.Helper()

	db, dbCleanup := setupCalendarTestDB(t)
	tracker, err := fairness.New(db)
	require.NoError(t, err)

	testScheduler := scheduler.New(&calendarTestConfigStore{
		parentA: "Alice",
		parentB: "Bob",
	}, tracker)

	tokenStore, err := database.NewTokenStore(db)
	require.NoError(t, err)
	require.NoError(t, tokenStore.SaveToken(&oauth2.Token{
		AccessToken: "token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}))
	require.NoError(t, tokenStore.SaveSelectedCalendar("primary"))

	tokenManager := token.NewTokenManager(tokenStore, &oauth2.Config{})
	fakeAPI := newFakeCalendarAPI(t, initialEvents...)
	server := httptest.NewServer(http.HandlerFunc(fakeAPI.handle))

	apiService, err := gcalendar.NewService(
		context.Background(),
		option.WithHTTPClient(server.Client()),
		option.WithEndpoint(server.URL+"/"),
	)
	require.NoError(t, err)

	service := New(&oauth2.Config{}, "https://app.example", "https://public.example", tokenStore, testScheduler, tokenManager)
	service.srv = apiService
	service.calendarID = "primary"
	service.initialized = true

	return service, fakeAPI, testScheduler, tracker, func() {
		server.Close()
		dbCleanup()
	}
}

func TestSyncScheduleRelinksExistingManagedEventBySourceURL(t *testing.T) {
	date := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	existingEvent := &gcalendar.Event{
		Id:      "existing-event",
		Summary: "Old summary",
		Start:   &gcalendar.EventDateTime{Date: date.Format("2006-01-02")},
		End:     &gcalendar.EventDateTime{Date: date.AddDate(0, 0, 1).Format("2006-01-02")},
		Source:  &gcalendar.EventSource{Title: constants.NightRoutineIdentifier, Url: "https://app.example"},
	}

	service, fakeAPI, testScheduler, tracker, cleanup := newSyncTestService(t, existingEvent)
	defer cleanup()

	assignment, err := tracker.RecordAssignment("Alice", date, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	require.NoError(t, tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, "missing-event"))

	assignments, err := testScheduler.GetAssignmentsInRange(date, date)
	require.NoError(t, err)
	require.Len(t, assignments, 1)

	require.NoError(t, service.SyncSchedule(context.Background(), assignments))

	updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, "existing-event", updatedAssignment.GoogleCalendarEventID)
	assert.Equal(t, 1, fakeAPI.eventCount())

	storedEvent := fakeAPI.event(t, "existing-event")
	assert.Equal(t, formatEventSummary(assignments[0]), storedEvent.Summary)
	assert.Equal(t, "https://app.example", storedEvent.Source.Url)
	assert.Equal(t, fmt.Sprintf("%d", assignment.ID), storedEvent.ExtendedProperties.Private["assignmentId"])
	assert.Equal(t, constants.NightRoutineIdentifier, storedEvent.ExtendedProperties.Private["app"])
}

func TestSyncScheduleRecreatesMissingManagedEvent(t *testing.T) {
	date := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)

	service, fakeAPI, testScheduler, tracker, cleanup := newSyncTestService(t)
	defer cleanup()

	assignment, err := tracker.RecordAssignment("Bob", date, false, fairness.DecisionReasonAlternating)
	require.NoError(t, err)
	require.NoError(t, tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, "missing-event"))

	assignments, err := testScheduler.GetAssignmentsInRange(date, date)
	require.NoError(t, err)
	require.Len(t, assignments, 1)

	require.NoError(t, service.SyncSchedule(context.Background(), assignments))

	updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAssignment.GoogleCalendarEventID)
	assert.NotEqual(t, "missing-event", updatedAssignment.GoogleCalendarEventID)
	assert.Equal(t, 1, fakeAPI.eventCount())

	storedEvent := fakeAPI.event(t, updatedAssignment.GoogleCalendarEventID)
	assert.Equal(t, formatEventSummary(assignments[0]), storedEvent.Summary)
	assert.Equal(t, "https://app.example", storedEvent.Source.Url)
	assert.Equal(t, fmt.Sprintf("%d", assignment.ID), storedEvent.ExtendedProperties.Private["assignmentId"])
	assert.Equal(t, constants.NightRoutineIdentifier, storedEvent.ExtendedProperties.Private["app"])
}
