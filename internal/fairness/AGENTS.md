# internal/fairness

Core scheduling algorithm and assignment history tracking.

## Purpose

Implements the fairness algorithm that determines which parent is assigned each night, tracks all assignments in the database, and provides statistics for the UI. Babysitter assignments are tracked separately and excluded from parent fairness calculations.

## Sub-packages

- `fairness/` — Tracker (database operations) + types + enums + `TrackerInterface`
- `fairness/scheduler/` — Schedule generation algorithm (`Scheduler`, `determineNextParent`, double-consecutive tracker)

## Key Types

### Tracker (`tracker.go`)

- `Tracker` — Reads/writes assignment records in SQLite.
- `Assignment` — A single night routine assignment (parent name, date, override flag, caregiver type, babysitter name, decision reason, Google Calendar event ID).
- `Stats` — Per-parent statistics (`TotalAssignments`, `Last30Days`).
- `MonthlyStatRow` — Monthly assignment count per parent.
- `AssignmentDetails` — Snapshot of both parents' stats at the time a decision was made (for transparency UI).

### Enums

- `DecisionReason` — Why a parent was chosen: `TotalCount`, `RecentCount`, `ConsecutiveLimit`, `Alternating`, `Unavailability`, `Override`.
- `CaregiverType` — `parent` or `babysitter`.

### Scheduler (`scheduler/scheduler.go`)

- `Scheduler` — Generates schedules using fairness rules; `determineNextParent` implements the cascade (see below).
- `Assignment` (scheduler-level) — Adds `ParentType` (A/B/Babysitter) on top of tracker's Assignment.
- Double-consecutive tracker (`scheduler_double_consecutive_test.go` covers it) — detects `AA BB` patterns during generation and swaps them; babysitter/override/unavailability nights break the run.

## Fairness Algorithm (`determineNextParent`)

Pre-check (before the cascade): unavailability — if one parent is unavailable on that weekday, assign the other; both unavailable is an error.

Decision cascade (first match wins):

0. **No prior parent assignments** — fewer (or equal) total assignments wins (`TotalCount`).
1. **TotalCount** — parent with fewer total assignments.
2. **ConsecutiveLimit** — totals tied and last parent had ≥2 consecutive assignments → force switch.
3. **RecentCount** — totals tied, no streak → parent with fewer last-30-day assignments.
4. **Alternating** — default: alternate from the last parent.

Parent-only entries for streaks and lastParent are derived via `parentOnly()`; babysitter nights are excluded from these calculations but kept in the full list for context.

## Babysitter Rules

- Babysitter assignments have `caregiver_type = 'babysitter'` and `override = true`.
- **Excluded from** `GetParentStatsUntil` and `GetLastParentAssignmentsUntil` — they don't affect fairness calculations.
- Always treated as **fixed** (override) in schedule generation.
- `UpdateAssignmentToBabysitter(id, name, override)` — Convert parent assignment to babysitter.
- `UnlockAssignment(id)` — Revert to parent type (clears override, sets `caregiver_type = 'parent'`).

## Key Interface (`TrackerInterface`, `interface.go`)

```go
RecordAssignment(parent, date, override, reason) (*Assignment, error)
RecordBabysitterAssignment(name, date, override) (*Assignment, error)
GetLastParentAssignmentsUntil(n, until) ([]*Assignment, error)  // parent-only
GetParentStatsUntil(until) (map[string]Stats, error)            // parent-only
GetAssignmentByDate(date) (*Assignment, error)
GetAssignmentsInRange(start, end) ([]*Assignment, error)
UpdateAssignmentParent(id, parent, override) error
UpdateAssignmentToBabysitter(id, name, override) error
UnlockAssignment(id) error
SaveAssignmentDetails(assignmentID, calcDate, parentA, statsA, parentB, statsB) error
GetAssignmentDetails(assignmentID) (*AssignmentDetails, error)
```

## Test Files

- `scheduler/scheduler_test.go` — Parent-only scheduling tests (overrides, recalculation, alternating).
- `scheduler/scheduler_babysitter_test.go` — Babysitter test suite (19 tests covering all algorithm paths).
- `scheduler/scheduler_double_consecutive_test.go` — Double-consecutive (`AA BB`) detection and swap tests.
- `scheduler/secheduler_long_test.go` — Long-period scheduling tests (14/30 days).
- `tracker_test.go`, `tracker_upsert_test.go` — Tracker CRUD and upsert behavior.
- `decision_reason_test.go` — Enum tests.

## Dependencies

- Uses: `internal/database`, `internal/config`, `internal/logging`
- Used by: `cmd/night-routine`, `internal/calendar`, `internal/handlers`
