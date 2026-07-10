# Lessons collected at the REPL's deletion

> Fifth in the series. Written 2026-07-10, the day the REPL and tokeniser were
> deleted (scaffolding retired on schedule, as
> [interface-sequencing.md](interface-sequencing.md) prescribed — git history
> holds the code). The grammar lessons live in
> [line-grammar-design.md](line-grammar-design.md); these are the lessons the
> *implementation* taught that hadn't been filed yet.

## 1. Inject your streams from day one

The `repl` struct took `io.Reader`/`io.Writer`s as fields, injected from main —
so its tests scripted input and captured output trivially, from the first day
it existed. `run()` read `os.Args` and wrote to `os.Stdout` directly — and
stayed untested its whole life; parameterising it is still on the debt list.
The irony marking the lesson: **the driver that died was the tested one.**
Whether a component is testable is decided at its birth, in its signature —
retrofitting injection costs more every week it waits.

## 2. No coincidental correctness between branches

The REPL's tokenise-error branch once "worked" only because the blank-input
branch beneath it happened to catch the same case — the error path printed its
message, fell through, and *borrowed* the reprompt, correct only because
`tokenise` happened to return zero tokens alongside its error. Two rules from
the incident:

- **Every branch owns its own exit.** A branch that works because of what the
  next branch does is a bug with a delay timer — the first refactor of either
  branch fires it.
- **If a caller relies on a property, the property is contract — document it.**
  "On error, returns no tokens" moved from accident to doc comment the day the
  dependence was noticed.

## 3. Placement is the permission system

`quit` never reached `execute` — it was intercepted in the drivers, so no
handler ever needed to reject it. `migrate` was intercepted in `run()` before
the command machinery was even wired, so the REPL *could not* invoke it: typing
"migrate" there hit plain "unknown command", with zero validation code written.
Capability boundaries enforced by **where words are dispatched** are stronger
than boundaries enforced by checks — a check can be forgotten; an absent code
path cannot be taken. (Same idea at type level: the command layer receiving an
opaque `[]Migration` it can count but not execute.)

## 4. Layer test suites should scale O(1)

The REPL's test file opened with a manifesto: the driver test is a *small,
fixed set of driver guarantees* — loop, prompts, quit, stream separation — and
deliberately does NOT enumerate commands, so it never grows as entities are
added. Per-command behaviour lives with the entity; routing with the router.
The sign a suite has this property: adding a feature grows exactly one test
file. The sign it doesn't: every feature touches a central test table that
somebody will eventually stop maintaining.

## Coda

The REPL existed for five weeks, was fully tested, taught two notes-files'
worth of lessons, and was deleted the week its successor's design was approved
— with the deletion *cheaper* than keeping it (its tokeniser, sentinels, and
loop-control vocabulary all left with it, shrinking main's grammar to argv
only). That is what a good scaffolding lifecycle looks like end to end.
