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

- **Repository pattern for persistence.** Storage lives behind interfaces; swap
  in-memory → file → DB by adding adapter implementations. The interface is
  introduced only when a _second_ backend exists, not preemptively.

- **"Store" vs "Repository" naming.** `Store` = a concrete persistence implementation
  (the adapter). `Repository` = the interface (the port the domain depends on). Kept
  deliberately distinct.

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

- **Service/handler layer earns its place with orchestration.** A dedicated service
  layer is justified when an operation spans multiple repositories or needs
  validation/orchestration (i.e. once boards/threads arrive). Until then, a thin
  transport over a single repository is honest; a pass-through service would be
  ceremony.

- **Testability seams:** clock injection via functional options (deterministic time
  in tests); injected `io.Reader`/`io.Writer` on transports (drive them with scripted
  input + captured buffers). Dependencies injected from `main` (the composition
  root); no globals.

- **Defer abstractions until the second use case** — interfaces, packages, and the
  service layer appear when there's a concrete reason, not on spec.

---

## Current state — as of 2026-06-09

- **`post` package:** `Post` entity, `Store` (in-memory map + counter + mutex,
  `Create`/`ByID`/`List`), `ErrNotFound`. Clock injection via `WithClock` (functional
  options). Full tests incl. `-race` and a deterministic clock test.
- **CLI REPL** (root `package main`): `cli` struct (`store`/`in`/`out`/`errOut` +
  methods), `parseInput`/`formatPost`/`formatList` as free functions. Commands:
  `post`, `get`, `list`, `quit`, plus empty/unknown handling. Errors routed to
  `errOut`.
- **No application handler layer yet** — the CLI currently calls the `Store`
  (repository) directly. The handler hub is still to be built.
- **Tooling:** pre-push hook (`gofmt` + `go vet` + `go test -race`); Delve for
  debugging.

---

## Roadmap — rings to fill (order not committed)

- **Persistence:** in-memory → file → database. Introduces the `Repository`
  interface + adapter package(s) (e.g. `postgres`). Transports/handlers depend on
  the interface.
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
