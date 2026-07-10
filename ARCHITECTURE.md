# Messageboard — Architecture & Design

This is an aspirational design doc, it's the **target architecture and the
reasoning behind the decisions**, not a description of the current code.

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

**The command/query handlers are the hub.** Every client (CLI, HTTP, web, TUI) calls
the handlers; the handlers call repositories; repositories hit the DB.
No transport talks to a repository or the DB directly.

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
  entities. A quote-aware tokeniser turns a REPL line into `[]string`; a pure
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
  "may fade" in-process user face, not the permanent admin one. (Retired in part
  2026-07-10: the REPL and its tokeniser were deleted when the TUI ring opened —
  scaffolding removed on schedule; git history holds the code,
  `learning-notes/` the lessons. One-shot argv is now the only text face, and
  `quit` left with the loop it controlled.)

- **REPL tokeniser implements shell word-splitting (decided 2026-07-03).** One-shot
  mode's tokens come from the shell, so the REPL must produce identical tokens for
  the same line or the two modes silently diverge. Hence shell semantics: quotes
  (single or double) toggle whether whitespace is literal — they never delimit
  tokens — so `gen"eral chat"` is one token, either quote type is literal inside
  the other, and `""` is a legal empty token. No escape sequences (quote the other
  quote instead). The alternative model — a quote starts a new token, as in CSV or
  string literals — was considered and rejected for exactly that divergence. An
  unterminated quote is an error: the named sentinel `ErrUnclosedQuotes`, so a
  future driver could catch it and prompt for a continuation line (bash-style
  multi-line input) instead of failing. (Retired with the REPL 2026-07-10; the
  grammar decisions stand for any future line-based input — see
  `learning-notes/line-grammar-design.md`.)

- **Uniform exact arity (decided 2026-07-03).** Every command takes an exact number
  of positional arguments; multi-word values are always quoted (`post create 1
"hello world"`). This supersedes the earlier identifier/free-text split (board
  names strict, post bodies / thread titles greedily joined), which was designed
  for a user-facing REPL and fell when the audience reframed: the TUI and web are
  the user faces, so the text grammar serves shell-literate admins and scripts,
  where exact arity is standard CLI behaviour. Bonus properties: a misplaced
  trailing flag becomes a loud usage error instead of silently joining into
  content; a command line copy-pastes verbatim between a script and a REPL
  session; habitual quoting protects apostrophes and literal quote marks in
  content. When flags arrive, arity applies to `fs.Args()` after flag parsing
  (flags before positionals — stdlib `flag` convention). Same shape as the
  repository-pattern note: a different justification replacing the old one, not
  a reversal of reasoning.

- **Migrations are explicit, versioned, and forward-only (decided 2026-07-03).**
  Schema changes no longer run at startup. `migrate` is a one-shot-only admin
  verb — the first of the planned admin/ops CLI, unavailable in the REPL by
  placement (it's intercepted in `run()` before the command machinery exists) —
  and every other invocation runs a read-only guard that refuses with an
  actionable error when the schema is behind, diverged, or newer than the
  binary. A `migration` ledger table (`name` PK + `applied_at`; chosen over
  `PRAGMA user_version` for name-based integrity checks and audit familiarity)
  records applied history, which must always be a **prefix** of the in-code
  `Migrations` list — append-only, names immutable once shipped. Each migration
  applies and records inside one transaction (both-or-neither; SQLite DDL rolls
  back too). Forward-only: no down-migrations, deliberately. Pre-ledger
  databases self-adopt — the original three creates are idempotent, so one
  `migrate` no-ops them and records everything. Migrations remain an in-code
  slice; file-based representation and fork-extension concerns are deliberately
  deferred (see open questions). Pre-release note (2026-07-04): the three
  initial creates were rewritten in place — `NOT NULL` on all value columns,
  non-empty `CHECK`s on `name`/`title`/`body` — and dev databases were reset
  rather than rebuilt by migration. Permitted because nothing had shipped; the
  append-only rule binds from first release.

- **Service/handler layer earns its place with orchestration.** A dedicated service
  layer is justified when an operation spans multiple repositories or needs
  validation/orchestration (i.e. once boards/threads arrive). Until then, a thin
  transport over a single repository is honest; a pass-through service would be
  ceremony. (Arrived 2026-07-04: `core/`, founded on the post+bump transaction —
  see current state. The ceremony line ages out when a second transport lands:
  pass-through reads then earn their place as the shared API every face calls.)

- **Testability seams:** clock injection via the adapter's unexported `now func()
time.Time` (set white-box in package tests — the functional-options version was
  dropped when persistence went SQL-only); fake repositories
  (`fakePostRepo`/`fakeBoardRepo`) implementing the consumer ports for command
  tests (no DB); a `:memory:` SQLite + `storage.Migrate` for the adapter contract
  suites and core's transaction tests. Dependencies injected from `main` (the
  composition root); no globals.

- **Defer abstractions until the second use case** — interfaces, packages, and the
  service layer appear when there's a concrete reason, not on spec.

---

## Current state — as of 2026-07-03

- **`storage` package:** DB infrastructure. `Open(path)` (open + ping),
  `Migrate(conn, []Migration)` (ledger-versioned: bootstraps the `migration`
  table, verifies the applied history is a prefix of the list, applies the
  pending tail one transaction per migration), `Pending(conn, []Migration)`
  (the read-only guard/diagnosis: returns the pending tail, or errors on
  divergent or newer-than-binary history; never creates anything), `Migration`
  type, `Migrations` (the central append-only schema list), and the
  `modernc.org/sqlite` driver blank-import (transitive — importing `storage`
  registers the driver). Connections are created here and injected into
  adapters. Tested (`-race`): recording, history-based skipping (proven with a
  non-idempotent rerun), rollback atomicity, virgin and grandfathered
  databases, divergence/newer refusal, fail-fast with named failures.
- **Domain + persistence packages `post` and `board`:** each holds its entity
  (`Post`/`Board`), its domain errors, and a SQLite adapter (`NewSQLite(db)`). Post:
  `Create`/`ByID`/`List`, timestamps stored as Unix epoch seconds via a `scanPost`
  helper, unexported `now` for white-box clock tests. Board: `Create`/`List`/`Delete`
  (delete uses `DELETE … RETURNING` to hand back the removed row), `name` is `UNIQUE`
  (→ `ErrDuplicateName`). The schema enforces what the code assumes: `NOT NULL`
  on all value columns, non-empty `CHECK`s on `name`/`title`/`body` (violations
  surface as raw constraint errors for now — friendly validation is deferred to
  the service layer). Each package has a black-box contract suite against a fresh
  `:memory:` DB, plus (post) a timestamp round-trip test.
- **Command system (`package main`):** entity-first and multi-entity.
  `commands.execute(tokens)` handles globals (`help`) then routes via `entityDispatch`
  → the per-entity command module (`postCommands`/`boardCommands`). Each module owns
  its consumer-side port (`postRepository`/`boardRepository`), its `dispatch`, its
  handlers, and its formatters. Every command takes exact positional arity
  (multi-word values arrive as one quoted token). Routing is case-insensitive;
  args and bodies keep their case. Post creation routes through a `postCreator`
  port satisfied by `core.Core`; reads keep their repo ports.
- **Application core (`core/`):** the command hub — `Core` holds the `*sql.DB`;
  `CreatePost` creates the post and bumps its thread to the post's creation
  time in one transaction, constructing tx-scoped adapters via the entity
  packages' `DB` interfaces (both `*sql.DB` and `*sql.Tx` satisfy them). Repo
  ports inside core are deliberately deferred — its tests want the real
  database, because the transaction is the subject. Tested (`:memory:`):
  returns/persists the post, bumped_at-equals-PostTime, missing-thread
  sentinel, rollback-on-bump-failure via a test-installed sabotage trigger.
- **One driver over the evaluator:** the **one-shot** path in `main`'s `run()` —
  `execute(os.Args[1:])` → stdout/stderr + exit code; bare invocation prints
  usage (and will become the TUI when it lands). The REPL and tokeniser were
  deleted 2026-07-10 (see the retired decision entries). `migrate` is
  intercepted in `run()` before the guard — admin verbs sit above the guard,
  user verbs below it — and every other invocation refuses, with a run-migrate
  message, if `storage.Pending` reports the schema behind. `main` is a thin
  error boundary over `run() error`.
- **Tests are layered to match the code:** adapter contract suites (in the entity
  packages); per-entity command tests at the `dispatch` level with fake repos
  (`post_commands_test.go`/`board_commands_test.go`); evaluator routing
  (`commands_test.go`); core's DB-backed transaction suite. Per-command behaviour
  lives with its entity, so adding an entity doesn't grow a central table.
- **Not yet:** `help` is a placeholder string. Reads and single-repo writes still
  call repos directly — they migrate into `core` as they earn it, or when the TUI
  wants the shared API. The `run()` wiring is hand-verified, not unit-tested —
  and it has accumulated real policy (usage before open, `migrate` above the
  guard, guard above everything), so parameterising `run(args, stdout, stderr)`
  is flagged as worth doing.
- **Tooling:** pre-push hook (`gofmt` + `go vet` + `go test -race`); Delve for
  debugging.

---

## Roadmap — rings to fill (order not committed)

- **Persistence:** ✅ done. Progressed in-memory → file → SQLite, then committed to
  SQLite alone (in-memory/file dropped). The `storage` infra package, the
  `post.SQLite` adapter, and the consumer-side `postRepository` port are in place.
- **CLI command system:** ✅ done. Entity-first, multi-entity commands
  (`commands` evaluator + `postCommands`/`boardCommands`) driven one-shot from
  argv, with layered tests. (Originally also a REPL driver sharing the same
  `execute`; retired 2026-07-10 when the TUI ring opened.)
- **Domain:** ✅ done. Boards, threads, and posts, plus the service/handler layer
  (`core/`, founded 2026-07-04 on the post+bump transaction — the hub the
  transports ring will grow against).
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

- **Third-party schema extension (only if the OSS story becomes real):** forks
  adding their own migrations must not splice into core's history — the
  append-only prefix invariant breaks on upstream merges. The mechanism would be
  namespacing: extension migrations get their own sequence (own list, own
  `source` column or table), applied after core's. Revisit the file-based
  migration representation then, not before.

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
- **Intermediates yes, ceremony no.** Capture results in named locals before
  returning (breakpoint-friendly for Delve), but no named result parameters and
  no error-relay branches that return the same pair on both paths.
- **Keep this doc at the decisions/why altitude;** let the code be the implementation
  detail.
