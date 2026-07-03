# Test quality — two axes, eight questions

> Third in the series, after [line-grammar-design.md](line-grammar-design.md)
> and [interface-sequencing.md](interface-sequencing.md). Distilled 2026-07-03
> from auditing the migration-runner suite in `storage/storage_test.go`.

Two different questions get conflated as "is this a good test": whether the
test **deserves to exist**, and whether it **earns trust** — proves what its
name claims. They're independent axes: a suite can be full of trustworthy tests
that shouldn't exist (redundant, tautological) or well-chosen tests that prove
nothing (vacuous, indiscriminate).

## Axis 1 — existence: which tests should there be?

1. **Test our code, not libraries or the language.** `database/sql` works;
   SQLite's transactions work. What's ours is the *composition* — e.g. the
   atomicity test doesn't test that SQLite rolls back DDL, it tests that
   `applyMigration` puts statement and recording inside *one* transaction: a
   two-`Exec` refactor passes every other test and fails this one.
2. **Test possible behaviour, not impossible edge cases.** If the scenario
   can't arise through any real path, the test documents fiction. (See the
   flexibility clause — synthetic *triggers* for real failure *classes* are
   fine.)
3. **No redundancy.** If one test pins the behaviour, let it be one. Redundancy
   is often *created by change*: `TestMigrateIsIdempotent` was load-bearing
   under the rerun-everything scheme and became a strict-subset shadow of the
   history-skip test the day the mechanism changed. Audits after refactors,
   not just before merges.
4. **Not tautological.** Building a structure and asserting it has the shape
   the setup gave it proves the assignment operator. The test must observe a
   *behaviour* — something the implementation could plausibly do differently.

## Axis 2 — trust: does the test prove what it claims?

1. **Name the property, not the scenario.** The name is what you read when the
   bar goes red; it should state the rule that broke ("mid-word quote joins
   into one token"). A name that can't commit to one property usually means the
   test is hiding two — split it.
2. **Assert every output, with the right comparator, exactly.** Value *and*
   error on every case; `slices.Equal` / `errors.Is`; counts are weak proxies
   for contents (a ledger with the right *number* of wrong names passed the
   count check).
3. **Discriminate — pass only for the right reason.** The strongest test in the
   suite makes wrong mechanisms *unable* to pass: rerun-safety proven with
   deliberately non-idempotent migrations can only succeed via history
   tracking. Ask: could a plausible wrong implementation pass this?
4. **Guard against vacuous passes.** Negative assertions ("X didn't happen")
   need a positive anchor proving the run happened at all — "good was
   recorded" alongside "broken wasn't"; "the error came from the recording
   step" before trusting that the rollback was exercised.

## The flexibility clause — legitimate near-misses

- **Synthetic trigger, real class.** A duplicate ledger name can't occur via
  the public path, but it's the only deterministic stand-in for "recording
  fails after the statement succeeds" (disk full, crash). The *behaviour*
  pinned is real; only the trigger is staged. Say so in the comment.
- **Identity as contract.** "Returns exactly its input" looks tautological
  until you list what the implementation could plausibly return instead
  (error, nil, empty — one draft did). If the identity is one behaviour among
  several, it's a real pin.
- **Relevant soon.** A test may pin behaviour nothing exercises *yet* but the
  next planned ring will. Keep, with a comment naming the future consumer.
- **Dated tests.** The mirror image: a test valid today (pre-ledger databases
  exist) that will drift into criterion-2 violation as the world moves on.
  Mark it as a deletion candidate in its comment so the future audit is cheap.

## Companion rule for test comments

Same contract-vs-narration test as doc comments ("could the implementation
change while this sentence must stay true?"), with one test-specific addition:
**explain any test data that is deliberately wrong-looking** — the missing
`IF NOT EXISTS`, the pre-inserted duplicate name — because that weirdness is
the test's whole meaning, and a well-intentioned cleanup will otherwise
destroy it without failing anything.
