# Messageboard — Architecture & Design

> A living design doc capturing the **target architecture and the reasoning behind
> the decisions** not a description of the current code.

---

## Vision

Hexagonal / ports-and-adapters. A pure domain **core** wrapped by thin transport
**adapters**. Every adapter funnels through the same internal API; nothing reaches
the database except through it.

```
               database
                   ↑
             repositories
                   ↑
        command / query handlers      ← the hub: the one internal API
                   ↑
   ┌───────────────┼───────────┬──────────┐
 in-proc CLI    JSON API    SSR web    TUI (SSH)      ← thin adapters
```

**The command/query handlers are the hub.** Every face (CLI, HTTP, web, TUI) calls
the handlers; the handlers call repositories; repositories hit the DB. No transport
talks to a repository or the DB directly. This keeps all business logic in one place
and every transport thin.

---

## Topology — two binaries

**1. `server`** — holds the core and exposes it through several faces:

- the **core**: entities + repositories + command/query handlers + DB
- an **HTTP listener**: the JSON API (`/api`) _and_ the SSR HTML web UI (`/`)
- a **Wish SSH listener**: serves the Bubbletea TUI to SSH connections
- **in-process CLI** subcommands: admin/ops (migrate, seed, bootstrap) and possibly
  local user commands

**2. `cli`** — a remote client. A pure **HTTP client** to the server's JSON API.
"Local" just means pointing at `localhost`; "remote" means another host — one flag
(the URL) decides. No embedded core.

All faces are thin adapters over the **same shared handlers**.

---

## Key decisions & rationale

- **Handlers are the hub.** Transports never touch repositories/DB directly. Note
  the word "handler" appears at two levels: _application_ command/query handlers
  (the shared core API) vs _transport_ adapters (e.g. a CLI's `handleGet`). The
  latter call the former.

- **Repository pattern for persistence — now SQL-only (2026-06-20).** Originally
  planned as a swappable ring (in-memory → file → DB). That collapsed: the
  in-memory and file backends served their learning purpose and were dropped;
  SQLite is the single backend (git history holds the rest). A persistence _port_
  still exists with one implementation, but it's declared at the **consumer** (the
  cli) as the seam it needs for testing — Go's "accept interfaces, define them at
  the consumer" — _not_ a producer-side abstraction for swapping backends. So the
  old "defer the interface until a second backend" rule gave way to "define the
  small interface the consumer needs": a different justification, not a reversal.

- **Adapter vs port naming (resolved 2026-06-20).** The concrete adapter is named
  for its mechanism — `post.SQLite` — and is constructed only at the composition
  root, the one place allowed to know the backend. The **port** is `postRepository`,
  an unexported interface declared at the consumer (`main`). "Store" as a type name
  was deliberately retired. Entity-prefixed so `threadRepository` / `boardRepository`
  can sit beside it. (Reasoning settled: mechanism-naming a concrete is fine because
  it's an _adapter_, only referenced where the backend is chosen — not a leak.)

- **The JSON API is a first-class, versioned contract** (`/api/v1/...`). It is stable
  so _any_ client — the remote CLI, a future SPA, third parties — can build against
  it. The web UI may iterate freely; the API must not break.

- **SSR HTML and the JSON API are siblings, not layers.** Both call the in-process
  core handlers directly, in parallel. The web layer must **not** call the JSON API
  over HTTP internally (no pointless network hop, no coupling to the API's HTTP
  shape). Layering web-over-API is only for when the web is a separate service.

- **Dual auth.** Token (header) auth for the JSON API (what the CLI wants);
  cookie/session + CSRF for the SSR web (what browsers want). Applied per route
  group. This is the main place co-hosting API + web adds complexity.

- **CLI split by purpose, not by mechanism:**
  - _user_ commands (post/read/list) → **HTTP client** — works local + remote,
    constrained to the public API.
  - _admin/ops_ commands (migrate, seed, first-run bootstrap) → **in-process** —
    direct core access, no server required, and deliberately _not_ exposed over HTTP.
  - The in-process _user_ CLI is partly scaffolding and may fade once HTTP/web/TUI
    exist. The in-process _admin_ CLI is permanent (migrations can't go over HTTP).

- **CLI command interface — multi-entity, REPL and one-shot share one core.**
  Entity-first noun-verb (`board create hiking`), one verb vocabulary across all
  entities. A quote-aware tokenizer turns a REPL line into `[]string`; a pure
  `execute(tokens) (output, error)` core dispatches entity → action →
  transport-handler. Both faces share `execute`: the **REPL** loops over it; the
  **one-shot** form (`messageboard board create hiking`, taken when `len(os.Args) >
  1`) calls it once and exits with a status code (stdout/stderr, non-zero on
  failure). `quit` is loop-control — a guard in each driver, never reaching
  `execute` (one-shot no-ops it); `help` is a real command inside `execute`. Flags
  use the stdlib `flag` package, per-command inside the handler (`ContinueOnError`,
  so a typo can't `os.Exit` the REPL), added only when a command needs options
  (e.g. `board create --description … --nsfw`). Consistency note: `execute` is
  shared across the two _cli modes_, not the cross-transport hub — these handlers
  are transport-level (they call repos directly), and the one-shot user CLI is the
  "may fade" in-process user face, not the permanent admin one.

- **Service/handler layer earns its place with orchestration.** A dedicated service
  layer is justified when an operation spans multiple repositories or needs
  validation/orchestration (i.e. once boards/threads arrive). Until then, a thin
  transport over a single repository is honest; a pass-through service would be
  ceremony.

- **Testability seams:** clock injection via the adapter's unexported `now func()
  time.Time` (set white-box in package tests — the functional-options version was
  dropped when persistence went SQL-only); a `fakeStore` implementing the
  `postRepository` port for transport tests (no DB); injected `io.Reader`/`io.Writer`
  on transports (scripted input + captured buffers); a `:memory:` SQLite +
  `storage.Migrate` for the adapter's own contract suite. Dependencies injected from
  `main` (the composition root); no globals.

- **Defer abstractions until the second use case** — interfaces, packages, and the
  service layer appear when there's a concrete reason, not on spec.

---

## Current state — as of 2026-06-20

- **`storage` package:** DB infrastructure. `Open(path)` (open + ping),
  `Migrate(conn, []Migration)` (runs in order, names the failing migration),
  `Migration` type, `Migrations` (the central ordered schema list for all entities),
  and the `modernc.org/sqlite` driver blank-import (transitive — importing `storage`
  registers the driver). Connections are created here and injected into adapters.
  Tested (`-race`): happy path, idempotency of the real schema, fail-fast,
  named-failure, empty list.
- **`post` package:** `Post` entity, `ErrNotFound`, and `SQLite` — the SQLite-backed
  adapter (`NewSQLite(db *sql.DB)`, `Create`/`ByID`/`List`). Timestamps stored as Unix
  epoch seconds; a `scanPost` helper round-trips them back to UTC `time.Time`.
  Unexported `now` field for white-box clock tests. A black-box contract suite runs
  against a fresh `:memory:` DB per subtest, plus a timestamp round-trip test.
- **CLI REPL** (root `package main`): `cli` struct (`posts`/`in`/`out`/`errOut`), where
  `posts` is the consumer-side `postRepository` port; `parseInput`/`formatPost`/
  `formatList` free functions. Commands `post`/`get`/`list`/`quit` + empty/unknown.
  Errors routed to `errOut`. Tested with a `fakeStore` (no DB).
- **No application handler layer yet** — the CLI calls the repository port directly.
  The handler hub is still to be built (justified once boards/threads bring
  cross-repository orchestration).
- **Tooling:** pre-push hook (`gofmt` + `go vet` + `go test -race`); Delve for
  debugging.

---

## Roadmap — rings to fill (order not committed)

- **Persistence:** ✅ done. Progressed in-memory → file → SQLite, then committed to
  SQLite alone (in-memory/file dropped). The `storage` infra package, the
  `post.SQLite` adapter, and the consumer-side `postRepository` port are in place.
- **CLI command system (next):** rework the single-entity REPL into the entity-first
  multi-entity command system (see Key decisions), with a shared REPL/one-shot
  `execute`. Unlocks board/thread commands and terminal one-shot use. Detailed build
  order is tracked outside this doc.
- **Domain:** add boards (and threads). New domain package(s) + likely the
  service/handler layer for cross-repository orchestration.
- **Transports:** Bubbletea TUI (a _second transport_ — motivates extracting the CLI
  into its own package and the multi-listener server); the HTTP server (JSON API +
  SSR web); then the separate remote `cli` client.
- **Build inward-out:** solidify the handler hub and the repository interface
  _before_ the outer transport/network rings, so the thin adapters slot on cleanly.

A given next move maps to a specific ring: a DB drives the repository interface +
adapter; boards drive a domain package + service layer; the TUI drives the second
transport (and the cli-as-package extraction).

---

## Open questions / deferred — as of 2026-06-09

- **Web rendering:** SSR HTML (leaning this for v0.1 — all-Go, one binary, simple)
  vs SPA + JSON API. The JSON API exists regardless; an SPA would just be another
  client of it.
- **Auth specifics:** token format, session store, CSRF strategy. Not needed until
  HTTP/multi-user. **Identity/users are absent so far** — Wish identifies users by
  SSH key; the web needs sessions. A user model is a future ring.
- **Service-layer timing:** exactly when to introduce it (likely alongside boards).
- **`Client` interface for user commands:** only needed if a _serverless standalone_
  user CLI is wanted (a direct impl alongside the HTTP impl). Current plan: split by
  purpose (user=HTTP, admin=in-process), so no `Client` interface yet.
- **Local CLI reach:** always-HTTP-to-`localhost` (leaning this — simpler) vs
  embedding the core. No standalone no-server _user_ CLI planned.

---

## Cross-cutting principles

- **Thin transports:** parse/translate → call handler → format. No business logic in
  adapters.
- **Inject dependencies** from the composition root (`main`); avoid globals.
- **Name for the call site** (avoid stutter; package context replaces prefixes).
- **Keep this doc at the decisions/why altitude;** let the code be the implementation
  detail.
