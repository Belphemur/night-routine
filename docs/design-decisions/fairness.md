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
