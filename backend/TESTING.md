# Backend Testing Policy

The backend test suite is risk-first. Tests should protect behavior that can
break user journeys, data integrity, authorization, concurrency, or production
capacity. Do not keep tests only because every CRUD endpoint or repository
method exists.

CRUD endpoint smoke tests and repository CRUD echo tests are not coverage goals.
Prefer one broad product journey or one persistence invariant over many tests
that only prove a request/row can round-trip.

## Maintainability

Split test files by behavior ownership, not by a hard line limit alone. A large
file is acceptable when it is a single harness or fixture, but it should be split
when it mixes unrelated public methods, product journeys, persistence invariants,
or reasons to change.

Guidelines:

- Unit, usecase, and domain tests should usually stay under 250-300 lines per
  behavior group.
- Integration tests may be 350-450 lines when they cover one repository
  invariant or transaction flow.
- E2E fixtures and server bootstrap files may be larger when they keep product
  journeys compact and avoid recreating a second API client layer.
- Load test setup, seed, and targeter files may be larger when they are harness
  code, but scenario assertions should stay in scenario-specific files.

When refactoring tests, first move fixtures and helpers without changing
assertions, then split scenarios, then run the focused package.

## Unit

Use unit tests for pure business rules, DTO mapping, error mapping, cache
helpers, validators, and small package utilities. Prefer mocks already provided
by the package under test, including `logkit.Noop()` for logger dependencies.

Run:

```sh
make test-unit
```

## Integration

Use integration tests when Postgres, Redis, storage, transactions, locks,
idempotency, or generated SQL behavior matter. Keep tests that cover races,
repository invariants, cross-table side effects, and non-trivial persistence
logic. Avoid one-method CRUD echo tests unless the query has a real risk.

Run:

```sh
make test-integration
```

## E2E

E2E tests cover complete product journeys, not every endpoint. Keep them
compact and self-contained in `e2e-test/`; avoid shared helper packages that
recreate a second API client layer. A good E2E test should cross multiple
boundaries: auth, team state, competition settings, scoring, files, websocket
events, or admin moderation.

Run:

```sh
make test-e2e
```

## Load

Load tests are not part of the default `make test` gate. Use short load tests
for local capacity regressions and full load tests before release-risk changes,
rate-limit changes, concurrency changes, websocket changes, or storage/cache
changes.

`make test-load-short` is the local representative profile. `make test-load`
is the full release/stress profile and includes spike, soak, endurance,
brute-force, and high-throughput scenarios.

Run:

```sh
make test-load-short
make test-load
```

## Benchmark

Benchmarks should track hot code paths and allocation-sensitive helpers.
Adding a benchmark is useful when performance is part of the contract or when a
refactor could change allocations in a shared path.

Run:

```sh
make bench
```
