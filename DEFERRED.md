# Deferred — known enhancements awaiting their triggers

Not a wishlist: every entry here was consciously deferred with a **named
trigger**. When a trigger fires, do the item (or consciously reject it) and
remove the row. Adding an item without a trigger is cheating — that's what
makes this a ledger rather than a graveyard.

## TUI package

| Item | Trigger |
| --- | --- |
| `truncate` moves from `boards.go` to a shared text helper | its second consumer (thread-list subject truncation) |
| Placeholder `renderThreadsPane` becomes a `threadsModel` component | the threads-pane build step |
| `currentView` (or a small view stack) alongside `focusArea` | the thread reader arrives — reader/composer are views you descend into with `esc`-back, not peer panes |
| `messages.go` for custom `tea.Msg` types | the data step (`Init` stops returning nil; boards load via `tea.Cmd`) |
| `paneChrome` struct bundling focus-reactive styles | a third focused/unfocused style pair (currently: pane, title) |
| Styles used by one pane move to that pane's file; shared styles promoted to `styles.go` | judged per style at its second consumer |
| Per-pane test files (`boards_test.go`, …) | a pane's tests outnumber the shared invariants in `tui_test.go` |
| lipgloss `Canvas`/`Layer`/`Compositor` for z-axis compositing | the help overlay (box over dimmed content) |
| `lipgloss.StyleRanges` for match highlighting | search (`/`) |
| `tea.View.Cursor` for real cursor support | the reply composer's textarea |
| Read the bubbles v2 upgrade guide before using `key`/`help` | the status-bar/keybar step (imminent) |
| Wide-rune-aware truncation | non-ASCII board/thread content ever mattering |

## Core / cross-layer

| Item | Trigger |
| --- | --- |
| View-shaped queries in core (thread list with reply counts, bump order, paging) | the TUI data step — design against the threads pane's known columns |
| Reads migrate behind core (pass-throughs earn their place as the shared API) | the TUI wiring real data — transport #2 consuming reads |
| Friendly validation errors (CHECK violations currently surface as raw driver text) | core's validation era — likely alongside the composer, where users first meet the errors |
| Board-name cap (24) as a shared constant | a third site needing the number (schema + sidebar sizing today; composer input limit is the likely third) |
| Verb matrix gaps (`post delete`, `board get`, `thread get`) | the TUI demanding them — build for the consumer, not the symmetry |
| `--no-bump`/sage on post creation | post-v1 feature; core skips the `Bump` call |

## Backend / infrastructure

| Item | Trigger |
| --- | --- |
| Parameterise `run(args, stdout, stderr)` for testability | the TUI settles main's final shape (bare invocation → TUI, `serve` verb) |
| Bare invocation launches the TUI (currently usage text) | the TUI is good enough to be the default face |
| `context.Context` on repository ports | the HTTP ring — retrofitting later touches every layer |
| SQLite `busy_timeout` + WAL pragmas | `messageboard serve` (Wish) — the first concurrent access |
| Migration-ledger checksum column (detect edited history) | ever actually needing edited-history detection |
| Schema-extension namespacing + file-based migrations | the OSS fork/extension story becoming real (see ARCHITECTURE open questions) |
| Delete `TestMigrateAdoptsPreLedgerDatabase` | no pre-ledger databases left in the wild (flagged in the test's own comment) |
