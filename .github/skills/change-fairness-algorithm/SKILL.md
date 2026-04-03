---
name: change-fairness-algorithm
description: >
  Guide for making changes to the fairness scheduling algorithm. Use this skill
  when you need to modify how parents are assigned to nights, change decision
  reasons, adjust the fairness cascade, or modify how babysitter/stats work.
  Covers all required touchpoints: scheduler, tracker, decision reasons, UI,
  tests, migrations, and design decisions.
argument-hint: "Description of the algorithm change, e.g. 'Add weighted scoring for weekend assignments'"
---

# Changing the Fairness Algorithm

Use this skill whenever you need to modify the fairness scheduling algorithm.
The algorithm has multiple touchpoints that **must all be updated together**.

## Architecture Overview

The fairness system has these key components:

### 1. Decision Reasons — `internal/fairness/decision_reason.go`

All possible reasons for a parent assignment. Current reasons:

- `Unavailability` — one parent was unavailable
- `Total Count` — parent with fewer total assignments (including babysitter shifts)
- `Consecutive Limit` — force switch after 2+ same-parent streak
- `Recent Count` — parent with fewer last-30-day assignments
- `Alternating` — default: alternate from last parent
- `Override` — manually overridden by user

**If you add/remove/rename a decision reason**, you must update all of the following.

### 2. Scheduler — `internal/fairness/scheduler/scheduler.go`

The core scheduling logic. Key function: `determineNextParent()`.

**Decision cascade (first match wins):**

1. **No prior parent assignments** → parent with fewer total (TotalCount)
2. **TotalCount** — parent with fewer total assignments
3. **ConsecutiveLimit** — when totals tied and 2+ same-parent streak, force switch
4. **RecentCount** — parent with fewer last-30-day assignments
5. **Alternating** — default: alternate from last parent

Helper functions:
- `parentOnly()` — filters assignment list to parent-only entries
- `otherParentOf()` — returns the other parent

The unavailability check happens in `determineParentForDate()` before `determineNextParent()` is called.

### 3. Tracker / Stats — `internal/fairness/tracker.go`

Database layer for assignment history and statistics.

Key functions:
- `GetParentStatsUntil(until, parentNames...)` — returns `TotalAssignments` and `Last30Days` per parent. **Babysitter nights count as +1 for both parents** (shift semantics). Pass parent names to seed the map for parents with zero assignments.
- `GetLastAssignmentsUntil(n, until)` — returns last N assignments (all caregiver types) for streak/gap detection.
- `RecordAssignment()` / `RecordBabysitterAssignment()` — persist assignments.

### 4. Tracker Interface — `internal/fairness/interface.go`

Defines `TrackerInterface` used by the scheduler. Any new tracker methods must be added here.

### 5. UI — Decision Reason Display

Decision reasons are shown in **two places** in the UI:

#### Calendar Grid (desktop + mobile)
- **Desktop**: `internal/handlers/templates/home.html` — `<span>` showing `.Assignment.DecisionReason`
- **Mobile**: Same file, JavaScript section — `day.assignmentReason` rendered as `<span>`

#### Details Modal — `internal/handlers/templates/home.html`
- The `explanations` JavaScript object maps each decision reason string to a human-readable explanation.
- **You MUST update this object** when adding/removing/renaming decision reasons.
- Search for `const explanations = {` in `home.html` to find it.

#### API Response — `internal/handlers/assignment_details_handler.go`
- `AssignmentDetailsResponse` struct includes `DecisionReason` field.

### 6. Database Migrations — `internal/database/migrations/sqlite/`

If you rename or remove a decision reason:
- Create a new migration (next sequential number) to UPDATE existing rows.
- Always provide both `.up.sql` and `.down.sql` files.
- Never modify existing migrations.

### 7. Tests

Tests are split across multiple files:
- `internal/fairness/scheduler/scheduler_test.go` — parent-only scheduling tests
- `internal/fairness/scheduler/scheduler_babysitter_test.go` — babysitter interaction tests
- `internal/fairness/scheduler/secheduler_long_test.go` — long-period balance tests (note: filename typo is historical)
- `internal/fairness/tracker_test.go` — tracker/stats tests
- `internal/fairness/decision_reason_test.go` — decision reason CRUD tests

### 8. Design Decisions — `docs/design-decisions/fairness.md`

Use the `record-decision` skill to document non-trivial algorithm changes.

## Checklist for Algorithm Changes

When making changes, follow this checklist:

- [ ] Modify `decision_reason.go` if adding/removing/renaming reasons
- [ ] Modify `determineNextParent()` in `scheduler.go` for cascade logic changes
- [ ] Modify `GetParentStatsUntil()` in `tracker.go` for stats calculation changes
- [ ] Update `interface.go` if adding new tracker methods
- [ ] Update the `explanations` object in `home.html` (search: `const explanations = {`)
- [ ] Create a DB migration if renaming/removing decision reasons from existing data
- [ ] Add/update regression tests in the appropriate test file
- [ ] Update `docs/design-decisions/fairness.md` via the `record-decision` skill
- [ ] Run `go test ./internal/fairness/...` to verify all tests pass
- [ ] Run `golangci-lint run` to check for issues

## Key Design Principles

- **KISS** — prefer the simplest correct solution
- **Babysitter = shift** — babysitter nights count as +1 for both parents in stats
- **TotalCount** is prioritized when totals are imbalanced — fairness correction comes first
- **ConsecutiveLimit** constrains long same-parent streaks once totals are tied
- **Decision reasons must be transparent** — users see them in the UI calendar and details modal
