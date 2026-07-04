# Patterns are solutions — check you have the problem first

> Fourth in the series. Distilled 2026-07-04 from the `bumped_at` design
> conversation, where the first instinct was to transplant a large production
> system's architecture (domain events, per-domain command handlers, thin
> cross-domain projections) onto a one-person messageboard.

## The law

Every architecture pattern is a solution to a specific problem, and it charges
for that solution whether or not you have the problem. Importing a pattern
without its problem means paying its costs for nothing — or worse, trading away
a guarantee the simple version gave you for free.

The case study: "posting bumps the thread" needs two writes to be atomic.

- **Domain events + listener** (the workplace pattern): exists because separate
  teams/services/databases *cannot share a transaction*. It trades atomicity
  for autonomy and buys eventual consistency, a bus, retries, idempotent
  listeners, and dropped-event handling. In one process with one SQLite file, a
  five-line transaction gives a strictly stronger guarantee at zero cost.
  Choosing events there is adopting eventual consistency as a *style*.
- **Thin local projection of the foreign entity** (a mini thread-repo inside
  the posts package): a necessity *between* services, where you can't import
  the other context's repository. In a monolith it means two packages writing
  one table — the same coupling you were avoiding, with extra ceremony and the
  coupling hidden instead of visible.

Neither pattern is wrong. Both are *between-contexts* tools applied *within* a
context. Don't pay distributed-systems prices when you're not distributed.

## The consistency boundary tells you where the domain is

The rigid version — "every command handler belongs to a domain, so packages
are domains" — mistakes entity persistence modules (`post/`, `thread/`) for
bounded contexts. The DDD-native correction: **invariants that must hold
transactionally define the aggregate/context boundary.** "Posting bumps the
thread, atomically" is exactly such an invariant — which is *evidence that
posts and threads are one domain*, not a coordination problem between two.
The whole messageboard is one bounded context; its entity packages are storage
organs, and the application layer (`core/`) is that one domain's use-case API.
Events are for crossing into *other* contexts — which don't exist yet.

## The sorting test

For any secondary effect of an operation: **if the secondary effect fails,
must the primary be undone?**

- Yes → it's part of the operation's definition. Same transaction, same
  handler. (The bump: a post without its bump silently corrupts ordering.)
- No → it's a reaction. Event/listener territory, eventual consistency is
  fine. (Future: notify subscribers, update a search index — you'd never roll
  back a post because an email failed.)

The workplace patterns aren't banned from this codebase; they're queued behind
the effects that actually fit them.

## Related small rule from the same session

Inside a transaction, prefer constraint-enforced invariants over
fetch-then-check: a pre-read ("does the thread exist?") races with concurrent
writes, while the foreign key enforces the same thing atomically with the
insert. Fetch-decide-write is the general command-handler template, but build
the fetch phase when a rule needs prior state, not before.

## When the big-system instincts switch back on

- A genuinely separate concern arrives (identity/users is the likeliest here):
  a real second context, and the boundary between contexts is where events,
  projections, and per-domain handlers earn their keep.
- The system actually distributes (second service, second database).
- Multiple teams own different parts, and autonomy starts outweighing
  atomicity.

Same reasoning as [interface-sequencing.md](interface-sequencing.md)'s "who
uses this?": architecture follows from the actual shape of the system and the
people building it — not from the shape of the biggest system you've seen.
