# Line-grammar design — lessons from the messageboard tokeniser

> Notes distilled from building the quote-aware tokeniser (2026-07-03) and the
> design discussion around it. The case study is this repo; the lessons are for
> the next CLI/REPL/TUI.

## The law

**Parsing complexity is proportional to how much structure the user may omit.**
Case-insensitivity, optional quotes, joined bodies, flexible arity — each is an
omission-affordance with a bill: spec, code, and tests. The spec is the expensive
part (the tokeniser was 40 lines of code and an evening of language design).
Strictness is prepayment by the user; for a RTFM developer audience it's a
legitimate currency.

## The ladder of line grammars

Ordered by parser cost. The jump that matters is to the "shell quoting" rung:
everything above it is *line arithmetic*; quoting is a *lexer* — state,
character classes, error cases, language-design decisions.

| Grammar                   | Parser               | Free-text args      | Human comfort |
| ------------------------- | -------------------- | ------------------- | ------------- |
| Fixed atoms only          | `strings.Fields`     | none                | constrained   |
| Rest-of-line trailing     | `strings.SplitN`     | one, must be last   | good          |
| Sentinel block / heredoc  | line loop            | multi-line          | good          |
| Shell quoting             | character-level lexer| any number, anywhere| best          |
| Length-prefixed (RESP)    | read N bytes         | arbitrary binary    | impossible    |

Note the U-shape: parser complexity is lowest at both extremes and peaks exactly
where human comfort peaks. Quoting is what you buy when humans must be able to
put any argument anywhere.

## Rest-of-line, in depth

First N arguments are atoms (whitespace-delimited); the final argument is
everything after the Nth delimiter, **verbatim**:

```go
parts := strings.SplitN(line, " ", 4) // "post create 1 hello   world, don't panic"
// parts[3] == `hello   world, don't panic` — untouched
```

Why it's cheap: the only reserved character is the newline, already spent
terminating the line. Trailing content is unconstrained *by construction* —
nothing left to collide with. No quotes, no escapes, no state.

### Lineage — each variant solves one problem

- **HTTP request line** (`GET /path HTTP/1.1`): pure fixed fields; header values
  are rest-of-line after `: `. Proof the pattern scales.
- **IRC** (`PRIVMSG #chan :hello world`): the `:` marker exists because IRC has
  *variable* middle arity — fixed arity per command needs no marker.
- **SMTP DATA**: rest-of-*stream* with a sentinel terminator (`.` on its own
  line) — and complexity creeps back as sentinel escaping (dot-stuffing).
  Heredocs (`<<EOF`) solve it better: the user picks a sentinel not in their
  content, so escaping never exists.
- **Redis**: runs *two* grammars — a space-split inline protocol for humans and
  length-prefixed RESP for programs (binary-safe, trivial parse). `redis-cli`
  layers quoting client-side. Strict grammar at the core; comfort at the edge.

### Where rest-of-line breaks (design checklist)

- Only **one** free-text argument per line, and it must be **last** (a name AND
  a bio in one command is inexpressible).
- Flags must precede positionals, non-negotiably.
- Newlines in content need an upgrade: sentinel block, or change channel
  entirely (`git commit` spawns `$EDITOR` rather than quoting a message).
- **The dual-mode trap** (what killed it here): rest-of-line only exists where
  raw lines exist. Argv invocation has no line left — the shell already split
  it — so a dual REPL + one-shot tool would speak two grammars. Messageboard's
  tokeniser wasn't bought by friendliness; it was bought by mode convergence.

## Modal REPLs: more code, not less

Type `board` to enter board mode, then `create`, then args — feels simpler,
isn't. Dispatch doesn't disappear; it smears across a state machine (session
state, mode-aware prompt, `back`/escape navigation, per-mode vocabularies).
Stateless `execute(tokens)` is table-testable; modal REPLs need scripted
sessions. And one-shot mode still needs the flat grammar, so modes are a second
interface, not a replacement. Modes earn their keep only to **amortise repeated
context** (Cisco config's `interface eth0`; hypothetically `thread 5` then many
post bodies) — a UX feature, never a build shortcut. The zero-parsing endpoint
of interactivity is a numbered menu wizard — which is a proto-TUI, so at that
point build the actual TUI.

## Lessons for next time

1. **Enumerate input channels before designing the grammar** (argv, raw lines,
   streams, forms). The grammar must be expressible in the *most constrained*
   channel supported. A REPL-only tool should take rest-of-line and be done in
   twenty minutes.
2. **Design the typed call first, grammar second.** The handler signature
   (`CreateThread(boardID, title)`) is the stable hub; every transport is just a
   recovery strategy for those types. Tokenising exists because a text line
   erases structure; a TUI never erases it, so it has nothing to parse.
3. **Price affordances in ladder rungs, not vibes.** "Users shouldn't need
   quotes for the body" is the difference between `SplitN` and a lexer.
4. **Strict grammar at the core, comfort at the edge.** Wire protocols stay
   machine-strict; quoting/prompting lives in the client (the Redis split).
5. **For rich content, change channel instead of enriching grammar** — editor
   spawn, heredoc, form, TUI. All cheaper than teaching a line grammar about
   paragraphs.
6. **Leniency that guesses is worse than strictness that teaches.** Greedy join
   silently absorbed a typo'd flag as content; exact arity turns the same
   mistake into a usage error. Dev tools should fail loudly and explain.

## The meta-lesson

"A REPL" is never just a loop — it's a **language**, and languages are priced by
their grammar, not their implementation. Catch that at the whiteboard, decide
which rung of the ladder the tool deserves, and most of the code writes itself.
