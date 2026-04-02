---
name: record-decision
description: >
  Record an architectural design decision in docs/design-decisions/. Use when a
  code change involves a non-trivial design choice, trade-off, or new pattern
  that future contributors should understand.
argument-hint: "Short decision title and the domain file, e.g. 'Paginated recent subtitles in streaming.md'"
---

# Record Design Decision

Use this skill to add or update design decisions following the project template.

## When To Use

- A code change introduces a meaningful architectural choice or trade-off
- An existing decision needs updating because the implementation changed
- The user asks to document why something was built a certain way

## Procedure

1. **Read the template** at `docs/design-decisions/TEMPLATE.md` to confirm the current structure.
2. **Identify the domain file** — decisions are grouped by domain (e.g., `fairness.md`, `calendar.md`, `handlers.md`). Pick the existing file that best fits. If no file matches, create a new one named after the domain.
3. **Read the target file** to see existing decisions and avoid duplicates.
4. **Write the decision** using this structure:

   ```markdown
   ## <Short Decision Title>

   **Decision**: One or two sentences describing **what** was decided.

   **Rationale**:

   - Bullet points explaining **why**
   - Focus on trade-offs, alternatives considered, and benefits

   **Implementation**: Where and how the decision is realised in the codebase. Reference file paths, interfaces, method names, and key types.
   ```

5. **Append** the new decision to the end of the file (before any trailing blank lines).
6. Keep each decision self-contained — a reader should understand it without reading the rest of the file.

## Rules

- One file per domain — group related decisions together.
- **Decision** = what was chosen. Keep it factual.
- **Rationale** = why it was chosen. Focus on trade-offs and benefits.
- **Implementation** = where it lives in code. File paths, interfaces, method names are welcome.
- Don't duplicate information already in `docs/architecture.md` or `docs-site/` — those describe behaviour, design decisions explain _why_.
