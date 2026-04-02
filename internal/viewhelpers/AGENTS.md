# internal/viewhelpers

Template view logic for the calendar UI.

## Purpose

Prepares assignment data for the calendar grid template. Converts a flat list of assignments into a structured week-by-week calendar suitable for HTML rendering.

## Key Types

- `CalendarDay` — A single day in the calendar grid (date, day number, whether it's in the current month, and its assignment if any).

## Key Functions

| Function | Purpose |
|----------|---------|
| `CalculateCalendarRange(refDate) (start, end)` | Returns the Monday–Sunday date range containing the full month |
| `StructureAssignmentsForTemplate(start, end, assignments) (monthName, weeks)` | Organizes assignments into a 2D grid (weeks × days) for the calendar template |

## Dependencies

- Uses: `internal/fairness/scheduler` (Assignment type)
- Used by: `internal/handlers/home_handler`
