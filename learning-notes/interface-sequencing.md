# Interface selection & sequencing — lessons from the messageboard REPL

> Companion to [line-grammar-design.md](line-grammar-design.md). That file is
> about pricing a grammar; this one is about whether the interface deserved a
> grammar at all. Written 2026-07-03, after realising the REPL was accidental
> scaffolding.

## What happened (the case study)

The plan was always TUI + web as the user faces. But the build started with a
REPL — not by decision, but by instinct: "the program should run as a continuous
process you're inside." That instinct was the *TUI trying to get out*; a REPL is
the cheapest inhabitable program, so that's what fell out. The one-shot mode was
added later, as an afterthought — inverted priority, since the one-shot is the
permanent face (scripting, admin) and the REPL is the fading one.

The sequencing error itself was cheap: `repl.go` + `tokeniser.go` are ~140 lines
and everything else (dispatch, handlers, formatters, ports) serves the one-shot
too. The real cost was **polishing scaffolding as if it were load-bearing** —
the tokeniser design sessions, shell-semantics decisions, and grammar-consistency
debates all served an interface with no long-term niche. The missing move wasn't
better analysis; it was asking "who uses this?" earlier.

## Who uses what (the matrix to draw on day one)

| Audience  | Interface                                  |
| --------- | ------------------------------------------ |
| End users | TUI (SSH) / web                            |
| Scripters | one-shot CLI                               |
| Admins    | one-shot for machine verbs (migrate, seed); TUI screens for human tasks |
| …the REPL | nobody, once the TUI exists                |

Ten minutes of this table in week one beats discovering it in week six.

## Why the one-shot is the natural scripting interface

Scripting is composition, and the shell composes **processes**: argv in, exit
code + stdout out. Control flow (loops, `if`, `set -e`, pipes, `$(…)`) lives in
bash; the tool just provides verbs. Piping commands into a REPL is *batching*,
not scripting — no per-command exit codes, no fail-fast, no control flow, and
prompt characters interleaved with output. (`expect(1)` exists because scripting
interactive interfaces is painful; if your tool needs it, the scripting
interface is missing.) Batch-over-one-connection has one real use — bulk seeding
without paying process+DB startup per command — and a dedicated admin `seed`
command beats it anyway.

## When a dedicated REPL earns its keep (the taxonomy)

1. **Startup amortisation** — JVM warmup, loading a model. Not: opening SQLite.
2. **Session state between commands** — an authed connection, an open
   transaction, a cursor. This is why psql / sqlite3 / redis-cli are REPLs.
3. **A language too rich for argv** — Python doesn't fit on a command line.
4. **No shell available** — embedded consoles, recovery environments.

No ticks → no REPL, because **bash already is a REPL for your one-shots**, with
history, line editing, and completion that a `bufio.Scanner` loop will never
have. A stateless one-line command set competes with readline and loses. The
calculus changes the day a session accumulates state worth keeping — that's the
re-justification trigger to watch for.

## Sequencing rules for next time

1. **Draw the audience × interface matrix before writing interface code.**
2. **Build the thinnest *permanent* transport first** (here: one-shot CLI — it
   exercises the entire core with near-zero interface tax), **then the primary
   face** (TUI/web). Scaffolding only where the permanent path can't reach.
3. **Gate any REPL behind the taxonomy above.**
4. **Scaffolding must stay cheap.** The moment you're polishing it — designing
   its language, debating its ergonomics — either promote it to product
   deliberately or stop. Scaffolding earns its keep by what it teaches and
   costs, not by surviving into the final building.
5. **Read your own architecture doc.** ARCHITECTURE.md had already marked the
   in-process user CLI as "scaffolding, may fade" — the plan knew; the build
   didn't consult it. A written decision only helps if the build order answers
   to it.

## Verdict on the case study

As product engineering: one-shot first, then TUI, no REPL — the counterfactual
is simply better. As a *learning* project (the stated goal): the tokeniser was
the highest-yield artefact in the repo, so the detour paid — but by accident.
Next time the detour should be chosen, not fallen into.
