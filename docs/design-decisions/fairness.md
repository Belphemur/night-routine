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

**Decision**: The `TotalCount` and `RecentCount` steps in the fairness cascade now check whether their chosen parent is the same as last night's parent. If assigning them would create a back-to-back consecutive and no recent unavailability caused the imbalance, the algorithm assigns the other parent instead with a new `ConsecutiveAvoidance` decision reason. A date-adjacency guard (`isLastAssignmentYesterday`) ensures the check only fires when the most recent parent assignment is from the calendar day immediately before the scheduled date — a babysitter night or any gap in parent assignments disables the back-to-back concern.

**Rationale**:

- At month boundaries (e.g. a 31-day month), one parent ends up with 16 assignments vs 15. Previously `TotalCount` would assign the same parent again to correct the imbalance, creating an unnecessary 2-in-a-row that users had to manually override.
- Not all imbalances need the same correction: unavailability-caused imbalances (e.g. "Bob can't do Wednesdays") are structural and need consecutive assignments to stay balanced; month-boundary imbalances self-correct through natural alternation.
- The `hasRecentUnavailability(lastAssignments, 2)` lookback distinguishes these cases: if either of the last 2 parent assignments was forced by `Unavailability`, the consecutive is allowed (the imbalance is real); otherwise `ConsecutiveAvoidance` fires.
- The same logic is applied to the `RecentCount` step to prevent last-30-day differences from causing unnecessary consecutives.
- `GetLastParentAssignmentsUntil` is parent-only (excludes babysitter assignments). Without the date-adjacency guard, the scheduler would incorrectly treat a non-adjacent parent assignment (separated by a babysitter night) as "yesterday" and fire ConsecutiveAvoidance when no true back-to-back exists.

**Implementation**: `determineNextParent()`, `hasRecentUnavailability()`, and `isLastAssignmentYesterday()` in `internal/fairness/scheduler/scheduler.go`. New `DecisionReasonConsecutiveAvoidance` constant in `internal/fairness/decision_reason.go`. Regression tests: `TestNoConsecutiveWithoutUnavailability` (30/31/59-day periods with no unavailability prove zero consecutives), `TestUnavailabilityExemptionAllowsConsecutive` (unavailability-caused imbalance still corrected), `TestConsecutiveAvoidanceAtMonthBoundary` (both branches: avoidance + unavailability exemption), `TestConsecutiveAvoidanceWithTotalCountImbalance` (subtests: no-conflict + conflict), and `TestBabysitterGapDisablesConsecutiveAvoidance` (date-adjacency guard with/without babysitter gap).

## Consecutive-avoidance gap detection uses all-assignment history (Option B)

**Decision**: The consecutive-avoidance logic fetches two separate histories from the tracker: `GetLastParentAssignmentsUntil` (parent-only, for fairness stats and streak counting) and `GetLastAssignmentsUntil` (all caregiver types, for babysitter-gap detection). The algorithm uses the full history to natively determine whether yesterday was a babysitter night, replacing the previous date-arithmetic proxy.

**Rationale**:

- The earlier approach used only the parent-only list and relied on date-arithmetic (`isLastAssignmentYesterday`) to infer a babysitter gap. This was an indirect workaround for a root cause: the algorithm did not know about babysitter nights at all.
- With `GetLastAssignmentsUntil` returning all assignments, `isLastAssignmentYesterday` can directly inspect `lastAllAssignments[0]`: if it was yesterday AND is a parent, a true consecutive exists; if it was yesterday AND is a babysitter, there is no consecutive concern — no date math required.
- `hasRecentUnavailability` now also scans `lastAllAssignments` instead of the parent-only list: a babysitter entry encountered during the scan naturally breaks the contiguous-chain scan, preventing an unavailability from a previous (babysitter-interrupted) parent chain from triggering the exemption incorrectly.
- The parent-only list is still used for streak counting (`ConsecutiveLimit`) and stats (`TotalCount`, `RecentCount`), which must exclude babysitter nights.
- Option B was preferred over the date-arithmetic guard (Option A) because it is semantically correct: the algorithm now has the full picture and makes decisions based on actual assignment types rather than inferred gaps.

**Implementation**: `GetLastAssignmentsUntil` added to `TrackerInterface` in `internal/fairness/interface.go` and implemented in `internal/fairness/tracker.go` (same query as `GetLastParentAssignmentsUntil` without the `caregiver_type` filter). `assignForDate()` in `internal/fairness/scheduler/scheduler.go` fetches both histories and threads `lastAllAssignments` through `determineParentForDate()` → `determineNextParent()`. `isLastAssignmentYesterday()` checks `CaregiverType == CaregiverTypeParent` on `lastAllAssignments[0]`. `hasRecentUnavailability()` scans `lastAllAssignments` and stops at any babysitter entry or date gap. Regression test: `TestBabysitterGapDisablesConsecutiveAvoidance` in `internal/fairness/scheduler/scheduler_babysitter_test.go`; tracker test: `TestGetLastAssignmentsUntil` in `internal/fairness/tracker_test.go`.
