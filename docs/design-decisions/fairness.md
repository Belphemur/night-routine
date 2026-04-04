# Design Decisions — Fairness Algorithm

## ConsecutiveLimit takes priority over RecentCount in fairness cascade

**Decision**: The fairness cascade order was changed from `TotalCount → RecentCount → ConsecutiveLimit → Alternating` to `TotalCount → ConsecutiveLimit → RecentCount → Alternating`. Streak prevention now runs before recent-count balancing.

**Rationale**:

- When a babysitter is inserted on a past day, the two parent assignments flanking it can form a same-parent streak (e.g. Antoine on Mar 29, babysitter on Mar 30, Antoine on Mar 31 — from the parent-only view that's Antoine, Antoine consecutive).
- Under the old order, `RecentCount` fired first. A 1-night difference in last-30-day counts (e.g. 14 vs 15) would assign the same parent again, preventing `ConsecutiveLimit` from ever triggering.
- Streak prevention is a stronger user expectation than a minor recent-count imbalance — nobody wants the same parent three nights in a row just because of a 1-night statistical edge.
- The `RecentCount` path still fires when there is no streak, so fine-grained balancing is preserved.

**Implementation**: `determineNextParent()` in `internal/fairness/scheduler/scheduler.go`. The consecutive-count loop and the `consecutiveCount >= 2` branch now execute before the `Last30Days` comparison. All existing tests pass unchanged because previous test setups never had a scenario where `RecentCount` and `ConsecutiveLimit` competed.

## Past non-override days after an override are recalculated

**Decision**: In `GenerateSchedule`, the "after an override → recalculate" check now runs before the "past → fixed" check, so past days between a babysitter override and today are recalculated instead of being frozen.

**Rationale**:

- When a user sets a babysitter on a day 2+ days in the past, intermediate past days (between the override and today) need to shift based on the new fairness state.
- Previously, the past-day check (`assignmentDayStr < currentDayStr → fixed`) ran first, locking those intermediate days before the override check could mark them for recalculation.
- Overrides are explicit user actions and should propagate their effects forward regardless of whether the affected days are in the past or future.

**Implementation**: The assignment classification loop in `GenerateSchedule()` at `internal/fairness/scheduler/scheduler.go`. The `earliestOverrideStr != "" && assignmentDayStr > earliestOverrideStr → continue` branch was moved above the `assignmentDayStr < currentDayStr → fixed` branch. Regression test: `TestBabysitterOnPastDayRecalculatesPastDaysBetweenOverrideAndToday` in `internal/fairness/scheduler/scheduler_babysitter_test.go`.

## ConsecutiveAvoidance removed — caused schedule imbalances

**Decision**: The `ConsecutiveAvoidance` logic (which prevented back-to-back same-parent assignments in `TotalCount` and `RecentCount`) was removed entirely. The `DecisionReasonConsecutiveAvoidance` constant, `hasRecentUnavailability()`, and `isLastAssignmentYesterday()` helpers were deleted. Migration 000018 replaces existing `'Consecutive Avoidance'` decision reasons with `'Total Count'` in the database.

**Rationale**:

- ConsecutiveAvoidance was designed to prevent unnecessary 2-in-a-row at month boundaries, but in practice it caused persistent imbalances: the algorithm would never allow the "payback" double night needed to restore fairness after one parent accumulated extra assignments.
- The unavailability exemption added complexity but didn't cover all edge cases — babysitter-related imbalances were not addressed.
- Removing it simplifies the cascade to: `TotalCount → ConsecutiveLimit → RecentCount → Alternating`. TotalCount now always picks the parent with fewer total assignments, so the behind parent may be assigned repeatedly until totals converge. Once totals are equal, ConsecutiveLimit (2+ streak force switch) caps further streak growth before the algorithm falls through to recent-count balancing.
- The simpler algorithm is easier to reason about and produces fairer schedules in real-world use.

**Implementation**: `determineNextParent()` in `internal/fairness/scheduler/scheduler.go`. Dead code removed: `hasRecentUnavailability()`, `isLastAssignmentYesterday()`, `DecisionReasonConsecutiveAvoidance`. Tests updated: `TestTotalCountCorrectsImbalanceAtMonthBoundary`, `TestTotalCountWithImbalance`, `TestBalanceOverLongPeriods`, `TestUnavailabilityImbalanceCorrectedByTotalCount`.

## Babysitter nights treated as shifts (+1 for both parents)

**Decision**: `GetParentStatsUntil()` now counts each babysitter assignment as +1 to both parents' `TotalAssignments` and `Last30Days`. A babysitter night is a "shift" — the night still happened, but neither parent did it, so both advance equally.

**Rationale**:

- Previously babysitter nights were fully excluded from parent stats. This meant converting a parent assignment to babysitter created an artificial imbalance: the parent who lost the assignment fell behind in counts, and the scheduler would over-correct.
- Treating babysitter as a shift keeps both parents' counts aligned. The relative difference from parent-only assignments is preserved, but the babysitter night doesn't create or widen a gap.
- This approach is simple (one additional query) and uses the existing `idx_assignments_caregiver_date` composite index for efficient lookups.

**Implementation**: `GetParentStatsUntil()` in `internal/fairness/tracker.go` runs a second `QueryRowContext` for babysitter count (with `COALESCE` for NULL safety) and adds the result to both parents' stats. Tests: `TestBabysitterShiftCountsForBothParentsTotalCount`, `TestBabysitterShiftCountsForBothParentsRecentCount`, `TestGetParentStatsUntil_BabysitterShiftCountsForBothParents`.

## Single assignment list simplifies the scheduler (KISS)

**Decision**: The scheduler fetches a single all-caregiver-type history via `GetLastAssignmentsUntil` and derives parent-only entries in-memory with `parentOnly()`. This replaced the previous two-query approach that fetched `GetLastParentAssignmentsUntil` and `GetLastAssignmentsUntil` separately and threaded both through the cascade.

**Rationale**:

- The two-list approach added a second tracker query per day and doubled the parameter count in `assignForDate` → `determineParentForDate` → `determineNextParent`, making the code harder to follow.
- Parent-only entries can be cheaply derived from the full list by filtering on `CaregiverType == Parent`. This gives the scheduler the full picture (babysitter gaps, date adjacency) from a single source of truth.
- Fetching 7 entries (instead of 5) ensures enough parent entries are available even when babysitter nights are interspersed.

**Implementation**: `parentOnly()` helper and single-list `determineNextParent(date, parentA, parentB, lastAssignments, stats)` in `internal/fairness/scheduler/scheduler.go`. `assignForDate()` calls only `GetLastAssignmentsUntil(7, date)`. `determineParentForDate()` passes one slice through to `determineNextParent()`. `GetLastParentAssignmentsUntil` was removed from `TrackerInterface` and `Tracker` as dead code — the single all-types query fully replaces it. `parentOnly()` is used for streak counting and lastParent detection. Regression tests updated to pass a single list.

## Double consecutive smoothing during schedule generation

**Decision**: A `doubleConsecutiveTracker` is maintained inline during the `GenerateSchedule` loop to detect and swap "double consecutive" patterns (AA BB → AB AB) where both runs are ≥ 2 and neither is caused by unavailability, override, or babysitter. The swap is performed immediately when the pattern is detected, and the database is updated via `RecordAssignment` upsert — no new tracker method was added.

**Rationale**:

- Back-to-back consecutive nights for both parents (e.g. Alice–Alice–Bob–Bob) feels unfair to users even when the cascade produces it correctly. Swapping the boundary gives a smoother alternating pattern (Alice–Bob–Alice–Bob).
- Inline detection during generation is preferred over post-processing because swaps are persisted via the existing `RecordAssignment` upsert as they are detected — no separate update method is needed, and the schedule slice stays consistent with the database at every step.
- Fixed (past/override) assignments reset the tracker so they are never modified. Only generated assignments with non-override, non-unavailability, non-babysitter reasons participate in swaps.
- Availability constraints are checked before each swap to ensure no parent is assigned to a day they are unavailable.
- The tracker state (previous and current consecutive runs) is reset after each successful swap or when a non-swappable assignment is encountered, keeping the algorithm simple and O(n).
- The current fairness cascade (TotalCount → ConsecutiveLimit → RecentCount → Alternating) rarely produces the AA BB pattern naturally, so this is primarily a safety net for edge cases and future algorithm changes.

**Implementation**: `doubleConsecutiveTracker` type with `observe()` method in `internal/fairness/scheduler/scheduler.go`. Instantiated in `GenerateSchedule()` and called after each generated assignment is appended. Helper functions: `isSwappable()` (excludes override/unavailability/babysitter), `isParentAvailableOnDate()` (checks day-of-week constraints). `consecutiveRun` struct tracks parent, start/end indices, and count. New `DecisionReasonDoubleConsecutiveSwap` in `internal/fairness/decision_reason.go`. UI explanation added to `explanations` object in `internal/handlers/templates/home.html`. Unit tests for the observe mechanism and integration tests through `GenerateSchedule` in `internal/fairness/scheduler/scheduler_double_consecutive_test.go`.
