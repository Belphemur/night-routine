# internal/fairness

Core scheduling algorithm and assignment history tracking.

## Purpose

Implements the fairness algorithm that determines which parent is assigned each night, tracks all assignments in the database, and provides statistics for the UI. Babysitter assignments are tracked separately and excluded from parent fairness calculations.

## Sub-packages

- `fairness/` — Tracker (database operations) + types + enums
- `fairness/scheduler/` — Schedule generation algorithm

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

- `Scheduler` — Generates schedules using fairness rules.
- `Assignment` (scheduler-level) — Adds `ParentType` (A/B/Babysitter) on top of tracker's Assignment.

## Fairness Algorithm (`determineNextParent`)

Decision cascade (first match wins):

1. **Unavailability** — If one parent is unavailable on that day of week, assign the other.
2. **TotalCount** — Parent with fewer total assignments wins.
3. **RecentCount** — If totals tied, parent with fewer last-30-day assignments wins.
4. **ConsecutiveLimit** — If both tied, and last parent had ≥2 consecutive days, force switch.
5. **Alternating** — Default: alternate from last parent.

## Babysitter Rules

- Babysitter assignments have `caregiver_type = 'babysitter'` and `override = true`.
- **Excluded from** `GetParentStatsUntil` and `GetLastParentAssignmentsUntil` — they don't affect fairness calculations.
- Always treated as **fixed** (override) in schedule generation.
- `UpdateAssignmentToBabysitter(id, name, override)` — Convert parent assignment to babysitter.
- `UnlockAssignment(id)` — Revert to parent type (clears override, sets `caregiver_type = 'parent'`).

## Key Interface (`TrackerInterface`)

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

- `scheduler_test.go` — Parent-only scheduling tests (overrides, recalculation, alternating).
- `scheduler_babysitter_test.go` — Comprehensive babysitter test suite (17 tests covering all algorithm paths).
- `secheduler_long_test.go` — Long-period scheduling tests (14/30 days).
- `tracker_test.go` — Tracker CRUD tests.
- `tracker_upsert_test.go` — Upsert behavior tests.

## Dependencies

- Uses: `internal/database`, `internal/config`, `internal/logging`
- Used by: `cmd/night-routine`, `internal/calendar`, `internal/handlers`
