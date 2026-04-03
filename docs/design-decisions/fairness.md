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

## ConsecutiveAvoidance prevents unnecessary back-to-back assignments

**Decision**: The `TotalCount` step in the fairness cascade now checks whether its chosen parent is the same as last night's parent. If assigning them would create a back-to-back consecutive and no recent unavailability caused the imbalance, the algorithm assigns the other parent instead with a new `ConsecutiveAvoidance` decision reason. The `RecentCount` step also avoids unnecessary back-to-back assignments, but always fires `ConsecutiveAvoidance` without the unavailability exemption — RecentCount only triggers when totals are already equal, so any last-30-day difference is minor and self-corrects. A date-adjacency guard (`isLastAssignmentYesterday`) ensures the check only fires when the most recent parent assignment is from the calendar day immediately before the scheduled date — a babysitter night or any gap in parent assignments disables the back-to-back concern.

**Rationale**:

- At month boundaries (e.g. a 31-day month), one parent ends up with 16 assignments vs 15. Previously `TotalCount` would assign the same parent again to correct the imbalance, creating an unnecessary 2-in-a-row that users had to manually override.
- Not all imbalances need the same correction: unavailability-caused imbalances (e.g. "Bob can't do Wednesdays") are structural and need consecutive assignments to stay balanced; month-boundary imbalances self-correct through natural alternation.
- The `hasRecentUnavailability(lastAssignments, 2)` lookback distinguishes these cases for `TotalCount` by scanning recent all-assignment history and only considering the contiguous window before the scheduled date. If it encounters a babysitter assignment or a date gap, it stops; if either of the two most recent qualifying assignments was forced by `Unavailability`, the consecutive is allowed (the imbalance is real); otherwise `ConsecutiveAvoidance` fires.
- `RecentCount` always avoids the consecutive without the unavailability exemption. This is intentional: RecentCount only fires when total assignments are equal, so the last-30-day difference is minor and will self-correct through alternation. Adding the exemption would add complexity for negligible benefit.

**Implementation**: `determineNextParent()`, `hasRecentUnavailability()`, and `isLastAssignmentYesterday()` in `internal/fairness/scheduler/scheduler.go`. `hasRecentUnavailability()` is used only for the `TotalCount` consecutive-avoidance exemption; the `RecentCount` branch always avoids back-to-back assignments. New `DecisionReasonConsecutiveAvoidance` constant in `internal/fairness/decision_reason.go`. Regression tests: `TestNoConsecutiveWithoutUnavailability` (30/31/59-day periods with no unavailability prove zero consecutives), `TestUnavailabilityExemptionAllowsConsecutive` (unavailability-caused imbalance still corrected), `TestConsecutiveAvoidanceAtMonthBoundary` (both branches: avoidance + unavailability exemption), `TestConsecutiveAvoidanceWithTotalCountImbalance` (subtests: no-conflict + conflict), and `TestBabysitterGapDisablesConsecutiveAvoidance` (date-adjacency guard with/without babysitter gap).

## Single assignment list simplifies the scheduler (KISS)

**Decision**: The scheduler fetches a single all-caregiver-type history via `GetLastAssignmentsUntil` and derives parent-only entries in-memory with `parentOnly()`. This replaced the previous two-query approach that fetched `GetLastParentAssignmentsUntil` and `GetLastAssignmentsUntil` separately and threaded both through the cascade.

**Rationale**:

- The two-list approach added a second tracker query per day and doubled the parameter count in `assignForDate` → `determineParentForDate` → `determineNextParent`, making the code harder to follow.
- Parent-only entries can be cheaply derived from the full list by filtering on `CaregiverType == Parent`. This gives the scheduler the full picture (babysitter gaps, date adjacency, unavailability scanning) from a single source of truth.
- Fetching 7 entries (instead of 5) ensures enough parent entries are available even when babysitter nights are interspersed.
- `isLastAssignmentYesterday` and `hasRecentUnavailability` inspect the full list directly; `parentOnly()` is used only where needed (streak counting, lastParent detection).

**Implementation**: `parentOnly()` helper and single-list `determineNextParent(date, parentA, parentB, lastAssignments, stats)` in `internal/fairness/scheduler/scheduler.go`. `assignForDate()` calls only `GetLastAssignmentsUntil(7, date)`. `determineParentForDate()` passes one slice through to `determineNextParent()`. `GetLastParentAssignmentsUntil` was removed from `TrackerInterface` and `Tracker` as dead code — the single all-types query fully replaces it. Regression tests updated to pass a single list.
