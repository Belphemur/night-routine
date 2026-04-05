# internal/viewhelpers

Template view logic for the calendar UI.

## Purpose

Prepares assignment data for the calendar grid template. Converts a flat list of assignments into a structured week-by-week calendar suitable for HTML rendering.

## Key Types

- `DisplayAssignment` — Presentation-layer DTO for calendar assignments. Decouples the UI from internal scheduler types so templates use plain strings.
- `CalendarDay` — A single day in the calendar grid (date, day number, whether it's in the current month, and its `*DisplayAssignment` if any).

## Key Functions

| Function | Purpose |
|----------|---------|
| `CalculateCalendarRange(refDate) (start, end)` | Returns the Monday–Sunday date range containing the full month |
| `StructureAssignmentsForTemplate(start, end, assignments) (monthName, weeks)` | Organizes `DisplayAssignment` DTOs into a 2D grid (weeks × days) for the calendar template |

## Dependencies

- Uses: none (standalone presentation types)
- Used by: `internal/handlers/home_handler`
