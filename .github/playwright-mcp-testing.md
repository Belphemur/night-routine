# Playwright MCP Testing Guide

This guide captures practical patterns that worked well while debugging the babysitter unlock flow.

## Goals

- Reproduce user flows exactly (desktop and mobile variants).
- Confirm both UI behavior and backend state changes.
- Avoid false negatives from modal transitions, duplicate labels, and hidden elements.

## Recommended Workflow

1. Start from a clean state:

- Navigate to `http://localhost:8888/`.
- Capture a snapshot before interacting.

2. Target deterministic elements:

- Prefer stable attributes like `#assignment-calendar td[data-date="YYYY-MM-DD"]` for date cells.
- Avoid ambiguous text selectors when both desktop and mobile cells exist.

3. Handle modal transitions explicitly:

- Wait for visibility class changes (`hidden` removed/added) before the next click.
- If transition overlays intercept pointer events, wait for animation to settle or click by id after visibility checks.

4. Validate network payloads, not only clicks:

- Attach request listeners and capture `postData()` for critical actions.
- Confirm the expected request path and method (`POST /unlock`).

5. Validate backend outcome, not only page URL:

- Read assignment state via API endpoints (for example `/api/assignment-details?assignment_id=...`) before and after action.
- Confirm fields changed as expected (caregiver type, babysitter name, decision reason).

## Lessons Learned from Unlock Testing

1. Request encoding must match server parsing.

- `FormData` sends `multipart/form-data`.
- The unlock handler uses `ParseForm`, so use URL-encoded payloads (`application/x-www-form-urlencoded`) for compatibility.

2. Duplicate labels can select the wrong target.

- Calendar cells can appear in both desktop and mobile DOM.
- Use scoped selectors (`#assignment-calendar ...`) or `:visible` when possible.

3. Transition overlays can block clicks.

- During modal close/open animations, one modal can intercept pointer events for another.
- Wait for unlock modal visible state before clicking confirm.

4. URL checks alone are insufficient.

- A redirect to `/` does not guarantee state changed.
- Always verify assignment details after the action.

## Playwright MCP Troubleshooting

- If MCP browser context is closed, re-open navigation and retry.
- If selectors time out, take a fresh snapshot and re-resolve element refs.
- If clicks fail due to interception, add explicit waits on modal state.

## Minimum Evidence for UI Fixes

For each UI bug fix verified with Playwright MCP:

1. Before state snapshot.
2. Action trace (which elements were clicked).
3. Request evidence (method/path/payload for critical actions).
4. After state evidence (API data or visible UI state).
5. Screenshots for before/after where relevant.
