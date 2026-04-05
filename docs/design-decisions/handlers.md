# Design Decisions — Handlers & Presentation

## DisplayAssignment DTO for Template Decoupling

**Decision**: Introduce `viewhelpers.DisplayAssignment` as a presentation-layer DTO so that HTML templates and view helpers never depend on the internal `scheduler.Assignment` type. The conversion from `scheduler.Assignment` → `DisplayAssignment` happens once, in the home handler's `generateCalendarData` method, at the boundary between business logic and presentation.

**Rationale**:

- **Separation of concerns** — templates should not reach into scheduler internals. `scheduler.Assignment` carries fields only meaningful to the scheduling engine (`GoogleCalendarEventID`, `UpdatedAt`), and its field types are domain-specific (`fairness.CaregiverType`, `fairness.DecisionReason`, `scheduler.ParentType`). Templates need plain strings.
- **Reduced coupling** — before the DTO, `viewhelpers` imported `internal/fairness/scheduler`, creating a transitive dependency chain from the UI layer deep into the fairness engine. With the DTO, `viewhelpers` has zero internal dependencies.
- **Simpler templates** — templates previously called `.ParentType.String` and cast `DecisionReason` to compare strings. With the DTO, all fields are already strings, so templates use direct `eq` comparisons.
- **DRY field mapping** — the scheduler already has a `convertTrackerAssignment` helper for its own `fairness.Assignment` → `scheduler.Assignment` mapping. The DTO conversion is a separate, thinner mapping that only extracts display-relevant fields.
- **Alternative considered** — making the scheduler return the DTO directly was rejected because other consumers (calendar sync, webhook handler) legitimately need the full `scheduler.Assignment` with `GoogleCalendarEventID` and `Override`.

**Implementation**:

- `internal/viewhelpers/assignment.go` — `DisplayAssignment` struct with plain `string` fields for `ParentType`, `CaregiverType`, and `DecisionReason`.
- `internal/viewhelpers/calendar.go` — `CalendarDay.Assignment` field changed from `*scheduler.Assignment` to `*DisplayAssignment`; `StructureAssignmentsForTemplate` accepts `[]*DisplayAssignment`.
- `internal/handlers/home_handler.go` — `generateCalendarData` performs the conversion: `scheduler.Assignment` → `DisplayAssignment` at the handler boundary before calling `StructureAssignmentsForTemplate`.
- `internal/handlers/templates/home.html` — template comparisons simplified from `{{eq .Assignment.ParentType.String "ParentA"}}` to `{{eq .Assignment.ParentType "ParentA"}}`.
