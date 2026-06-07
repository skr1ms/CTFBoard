# CTFd vs AstroCTFb: backend/core analysis

Status: draft, living document
Date: 2026-06-07
Scope: backend/core product behavior, API contracts, persistence, high-load readiness, testing.
Excluded: frontend UX, CTFd plugin loading/extension system, themes, managed hosting, per-instance/deployment customization.

## Executive verdict

AstroCTFb backend/core does not look like a bad UX/product idea. The current `solo_only` / `teams_only` / `flexible` model is technically coherent and is implemented through competition settings, solo wrapper teams, team-switch guards, and submission eligibility checks. However, `flexible` has no clear target event format right now and should be treated as a removal target, not a feature to keep refining.

The intended participation model should be simplified to `teams_only` and `solo_only`. `teams_only` remains the conservative default for serious team CTFs, and `solo_only` covers individual events. Removing `flexible` should reduce fairness ambiguity, simplify team flows, and remove a mixed-mode branch that currently lacks a strong product use case.

AstroCTFb is already stronger than CTFd in several high-load/backend areas: Go service boundaries, OpenAPI-first contracts, explicit transactions, row/advisory locks, Redis/local cache paths, freeze-aware scoring, WebSocket/SSE events, and race/load/e2e gates. CTFd remains more mature as a broad CTF product model: richer flag/challenge logic, requirements reveal semantics, unlock/resource semantics, organizer-facing workflows, and a wider established API surface.

## Methodology

Scoring:

- `+2`: strong AstroCTFb advantage
- `+1`: AstroCTFb advantage
- `0`: parity or comparable outcome
- `-1`: CTFd advantage, manageable AstroCTFb gap
- `-2`: serious AstroCTFb gap for the stated positioning
- `N/A`: excluded from this analysis
- `?`: insufficient evidence

Each row includes:

- `Impact`: high / medium / low
- `Confidence`: high / medium / low
- `Gap type`: product / architecture / high-load / security / ops / testing / intentional divergence

Evidence IDs:

- `A*`: AstroCTFb evidence
- `C*`: CTFd evidence

Decision statuses:

- `accepted`: intentional divergence; no implementation work for this comparison.
- `done`: implementation or contract work is complete and covered by the stated exit criteria.
- `clarify`: behavior exists or mostly exists, but needs a documented contract or parity test.
- `implement`: proven gap inside the current backend/core scope.
- `split-prd`: too broad for this analysis; create a focused PRD before implementation.
- `out-of-scope`: explicitly excluded from this comparison.

## Comparison matrix

| Area | Score | Impact | Confidence | Gap type | Verdict |
| --- | ---: | --- | --- | --- | --- |
| Participation model | +1 | High | High | Product / simplification target | AstroCTFb has explicit solo/team modes and currently also has `flexible`; decision is to remove `flexible` and keep only clear event formats. |
| Competition lifecycle | +1 | High | High | Product | AstroCTFb models start/end/freeze/pause and team-switch policy directly; CTFd has mature setup/runtime configs but less explicit mixed-mode logic. |
| Challenge and flag model | -2 | High | High | Product | CTFd is deeper: separate flags, multiple flag classes, `any/all/team` logic, partials. AstroCTFb is narrower: one hash or regex model per challenge. |
| Requirements reveal semantics | -1 | Medium | High | Product | AstroCTFb has prerequisite edges; CTFd additionally supports anonymized/preview locked challenge reveal behavior. |
| Hints, solutions, unlocks | -1 | Medium | High | Product | AstroCTFb has hints/unlocks and one solution row; CTFd has explicit unlock polymorphism and solution state. |
| Submission pipeline and abuse controls | +2 | High | High | High-load / security | AstroCTFb submit path is more concurrency-aware and security-aware; CTFd has mature statuses and KPM/rate-limit behavior. |
| Dynamic scoring and freeze | +1 | High | High | High-load | Both support dynamic scoring/freeze. AstroCTFb stores `points_at_solve` and uses freeze-aware cache keys; CTFd has very mature tie-break semantics. |
| Scoreboard tie-break semantics | 0 | High | Medium | Product | Both have serious scoring paths, but CTFd's tie-break-by-achievement-time/ID behavior is clearer in code and docs. AstroCTFb should document its exact tie-break contract. |
| Users, teams, bans, brackets | +1 | High | High | Product / security | AstroCTFb has stronger ban propagation and solo-team semantics; CTFd has mature public hidden/banned/team field behavior. |
| Custom fields | 0 | Medium | High | Product | AstroCTFb now supports field `description`, `public`, and `editable` metadata with user/team own/public value visibility and editable profile patches; CTFd remains broader on JSON value semantics. |
| Auth, API tokens, rate limits | +1 | High | High | Security | AstroCTFb has hashed API tokens, Redis login lockout, and fine-grained route buckets; CTFd has mature email, registration code, and ratelimited auth flows. |
| Visibility knobs | 0 | Medium | High | Product | AstroCTFb now has separate challenge, score, account, and registration visibility parameters plus registration code/user-cap policy; CTFd remains the mature broad product baseline. |
| Admin API/content surface | 0 | Medium | Medium | Product | Both cover many admin/content areas. CTFd's API namespaces are broader; AstroCTFb has strong OpenAPI discipline and backup/storage/admin usecases. |
| Admin statistics | +1 | Medium | High | Product / ops | AstroCTFb now has solve percentages, distributions, time series, solve matrix, and an opened/attempted/solved admin funnel; CTFd remains the broader mature product baseline. |
| Data portability | 0 | Medium | Medium | Ops / product | Both have export/import concepts. AstroCTFb has versioned backup/import with file safety checks; CTFd has a mature ZIP lifecycle and migration/import status model. |
| API contract discipline | +2 | High | High | Architecture | AstroCTFb is OpenAPI-first with generated types/server/spec/client and codegen checks. CTFd has a broad RESTX API but less contract-first structure. |
| Architecture boundaries | +2 | High | High | Architecture | AstroCTFb has explicit Clean Architecture boundaries and handler rules. CTFd is mature but model/config/plugin concerns are more tightly coupled. |
| Realtime events | +2 | Medium | High | High-load | AstroCTFb has WebSocket/SSE contracts, async broadcaster, and WS load tests. CTFd core is less realtime-oriented. |
| Testing/load gates | +2 | High | High | Testing / high-load | AstroCTFb has unit/integration/e2e/race/load targets and hot-path benches. CTFd has mature regression tests, but not the same high-load positioning. |
| Multi-DB compatibility | -1 | Low | Medium | Product / ops | CTFd historically supports multiple SQL backends. AstroCTFb is intentionally PostgreSQL-first with pgx/sqlc style. This is probably acceptable for high-load. |
| Plugin ecosystem | N/A | Low | High | Excluded | Excluded by request. Mention only when a CTFd capability lives in plugin files but is core product behavior. |
| Frontend UX/themes | N/A | Low | High | Excluded | Excluded by request. |
| Per-instance/deployment customization | N/A | Low | High | Excluded | Excluded by request. Infra may be mentioned only as backend/high-load readiness. |

## Decision register

| Area | Status | Priority | Decision | Exit criteria | Target |
| --- | --- | --- | --- | --- | --- |
| Product positioning | accepted | Must | Position AstroCTFb as a high-load CTF backend/core, not a CTFd clone. | README/docs avoid whole-product superiority claims while frontend/plugins/per-instance customization are excluded. | This PRD / product docs |
| Remove flexible mode | done | Must | Remove `flexible` fully; keep only `solo_only` and `teams_only`. | Domain enum, DB constraints, OpenAPI schemas, setup/admin UI, generated frontend schema, guards, and tests no longer accept or branch on `flexible`. | `remove-flexible-mode` |
| Scoring and freeze | done | Must | Document exact public scoring contract instead of relying on SQL behavior. | Contract covers score sources, freeze cutoff, tie-break order, bracket scope, awards, and hidden/banned filtering. | `scoring-contract-and-fairness` |
| Challenge answer model | done | Must | Keep current single hash/regex model for v1; defer multi-flag groups, `any/all/team`, partial submissions, and migration strategy to a separate v2 implementation PRD. | `PRD/challenge-answer-model-v2.md`, backend docs, and OpenAPI descriptions document the v1 contract and explicitly avoid schema/runtime changes. | `challenge-answer-model-v2` |
| Solution and unlock semantics | done | Must | Use Astro-native solution visibility states instead of CTFd-compatible unlock APIs. | Backend stores explicit solution states in the squashed base schema, gates player solution/writeup-file access through `writeup_enabled`, solve status, and competition end state; legacy admin saves preserve current state; paid/free solution unlock records are deferred. | `solution-unlock-policy` |
| Visibility and registration | done | Should | Keep Astro-native visibility/registration policy instead of cloning CTFd config surfaces. | Challenge, score, account, registration visibility, registration code, user cap, admin override, OAuth new-account behavior, and password policy are covered. | `visibility-registration-policy` |
| Admin statistics funnel | done | Should | Use an Astro-native admin progression funnel instead of CTFd-compatible APIs. | OpenAPI/admin route exposes opened, attempted, solved rows/cells for challenges plus top teams/users; SQL respects hidden/banned filtering and freeze/live admin mode; tests cover funnel states and cache invalidation. | `statistics-funnel` |
| Data portability | done | Should | Keep Astro-native backup/import/export only; CTFd-compatible migration is explicitly not a goal. | Contract covers archive format, versioning, file validation, restore order, import status, async import jobs, and Astro-native compatibility. | `data-portability` |
| Namespace/API parity | clarify | Should | Use CTFd namespaces as a product checklist, not an architecture template. | Table below is maintained as equivalent / covered differently / missing / intentionally omitted / out-of-scope. | This PRD |
| Notifications and tracking | done | Should | Keep AstroCTFb realtime advantage and use Astro-native notification/tracking contracts, not CTFd-compatible APIs. | Global notifications have list/count timestamp-cursor reads, user inbox has unread counts, admins can deliver personal or team fan-out notifications, tracking cleanup reports deleted row counts, admin funnel consumes challenge-open tracking, and challenge-open writes are idempotent per user/challenge. | `notifications-tracking-parity` |
| Challenge metadata envelope | done | Could | Add Astro-native challenge metadata without cloning CTFd challenge logic. | `attribution` and nullable `next_id` are stored, exposed, validated, imported, exported, and redacted for locked/anonymized responses. Existing `connection_info` and `position` remain in the challenge contract. | `challenge-metadata-envelope-v1` |
| Challenge topic taxonomy | done | Could | Add Astro-native admin-only topics without cloning CTFd route semantics. | Normalized topics and challenge-topic assignments are stored, managed through admin routes, preserved in JSON backup import/export, and kept out of player challenge responses for now. | `challenge-topics-v1` |
| Solve share links | done | Could | Add Astro-native signed solve share links without cloning CTFd share APIs. | Authenticated verified team users can create a signed public solve link; public resolve returns escaped HTML/OG preview and hides invalid MACs, disabled shares, hidden challenges, hidden/banned teams, banned users, and soft-banned solve rows. | `share-links-v1` |
| Custom field JSON values | done | Could | Add typed Astro-native custom field values without cloning CTFd field-entry APIs. | `custom_fields` request/response values are typed by field definition: text/select strings, integral numbers, booleans, and arbitrary JSON values. Storage remains canonical string-backed for backup/import stability. | `custom-field-json-values` |
| Unlocks read model | done | Could | Add an Astro-native unlock read contract without adding CTFd-compatible solution unlock writes. | `/admin/unlocks` returns a generic unlock shape backed by existing hint unlock records; v1 keeps solution access under explicit visibility states and defers paid/free solution unlock charging. | `unlocks-v1` |
| Plugins, themes, per-instance customization | out-of-scope | N/A | Do not analyze or implement in this pass. | Mention only when a CTFd core capability lives in a plugin file. | Excluded |

## CTFd namespace parity

| CTFd namespace / surface | AstroCTFb status | Decision |
| --- | --- | --- |
| `challenges` | covered differently | Stronger safe submit path; v1 metadata envelope now covers `attribution`, `connection_info`, `next_id`, and `position`. Remaining gap is richer challenge `logic` plus partial/multi-flag behavior. |
| `flags` | missing / split-prd | Current one hash or regex per challenge is intentional v1 simplicity; advanced flags belong in `challenge-answer-model-v2`. |
| `hints` | equivalent / clarify | Hint CRUD/unlocks exist; compare paid/free cost semantics and duplicate unlock behavior. |
| `solutions` | covered differently / done | AstroCTFb has one solution row with explicit visibility states; CTFd also supports separate solution unlock behavior. |
| `unlocks` | covered differently / done | AstroCTFb exposes an Astro-native admin unlock read model backed by hint unlocks; paid/free solution unlock writes remain deferred. |
| `tags` | equivalent | Basic tag surface exists. |
| `topics` | covered differently / done | AstroCTFb now has admin-only normalized topics separate from tags, with challenge assignments and backup JSON round-trip. No player-facing topic discovery/filter API yet. |
| `awards` | equivalent / clarify | Awards exist; scoring contract must state award freeze and tie-break interaction. |
| `submissions` | equivalent / clarify | Strong consistency path exists; gap is no `partial` status until answer model v2. |
| `scoreboard` | covered differently | AstroCTFb has cached/freeze-aware scoreboard; document tie-break contract explicitly. |
| `statistics` | covered differently / done | AstroCTFb has solve percentages, distributions, time series, solve matrix, and an Astro-native opened/attempted/solved progression funnel. |
| `teams`, `users`, `brackets` | equivalent / clarify | AstroCTFb is stronger on solo teams and ban propagation; CTFd is richer on profile/custom public metadata and registration controls. |
| `configs`, `/configs/fields` | covered differently / clarify | AstroCTFb has settings/fields and visibility params; compare cache invalidation, registration fields, and admin update surface. |
| `files` | equivalent / clarify | AstroCTFb storage is stronger operationally; data-portability PRD should define archive and restore behavior. |
| `notifications` | covered differently / done | AstroCTFb has persisted global notifications, persisted per-user inbox notifications, global count with `since_created_at` timestamp cursor, unread user count, personal delivery, team fan-out delivery, and WS/SSE for global events. |
| `comments`, `pages` | equivalent | Covered enough for backend/core comparison. |
| `tokens` | covered differently | AstroCTFb API tokens are hashed and expiring; CTFd token API is broader as mature user setting surface. |
| `exports` | covered differently / clarify | AstroCTFb backup/import is versioned and safety-aware; CTFd has mature raw CSV/ZIP export and background import lifecycle. |
| `shares` | covered differently / done | AstroCTFb has stateless signed solve share links with public HTML previews and dynamic `social_shares_enabled` revocation. It intentionally does not provide CTFd-compatible share semantics. |
| Plugin and theme surfaces | out-of-scope | Excluded by request. |

## Detailed findings

### 1. Participation model: solo / team, remove flexible

CTFd supports a global `users` or `teams` mode. The enum has only those two values, and helpers resolve account URLs/models from the global `user_mode` config.

AstroCTFb supports `solo_only`, `teams_only`, and `flexible` at the competition level. The DB enforces allowed modes and team size constraints. Solo players are represented as solo wrapper teams, and submission eligibility checks enforce solo/team mode, banned state, and minimum team size.

Difference:

- CTFd chooses one account model per event.
- AstroCTFb currently can run solo and team participation inside one backend model, but this mixed event format does not have a clear product use case.
- The target model should keep competition-level `solo_only` and `teams_only`, while removing `flexible`.

AstroCTFb plus:

- `solo_only` and `teams_only` are clear event formats and map well to individual qualifiers and serious team competitions.
- Solo wrapper teams remain useful even after removing `flexible`, because they keep scoring unified for `solo_only` without splitting the solve model.

AstroCTFb minus/risk:

- `flexible` adds fairness ambiguity and extra branches across domain, OpenAPI, setup/admin UI, frontend generated schema, team flows, guards, and tests.
- Keeping `flexible` as a deprecated or hidden mode would preserve complexity without a concrete event format.

Recommendation:

- Keep `teams_only` as the recommended default for international team CTF finals and corporate team events.
- Keep `solo_only` for individual qualifiers.
- Remove `flexible` completely instead of documenting a fairness policy for a mode with no clear application.

Evidence: `A2`, `A3`, `C1`, `C2`.

### 2. Competition lifecycle and visibility

CTFd has mature setup/runtime config for visibility, registration, scores, and event timing. It separates challenge visibility, score visibility, account visibility, and registration visibility.

AstroCTFb models competition lifecycle directly: start time, end time, freeze time, pause state, public flag, global flag regex, mode, team switching, team-size limits, and freeze-after-end behavior.

Difference:

- AstroCTFb is stronger on competition lifecycle as a backend state machine.
- AstroCTFb now has separate visibility parameters for challenge, score, account, and registration visibility.
- CTFd is still a useful UX reference for mature registration workflow controls, but AstroCTFb now implements the key backend registration controls without aiming for CTFd compatibility.

AstroCTFb plus:

- Pause/freeze/submission windows are explicit domain behavior.
- Roster changes are guarded by competition status and `AllowTeamSwitch`.

AstroCTFb minus/risk:

- Visibility policy exists and registration now has a backend contract: `registration_open=false` closes self-service registration; `registration_visibility=private` hides/closes self-service registration; `registration_code` is an optional password-registration gate; `max_users` caps participant user accounts; admin create is an intentional override.
- OAuth new-account registration respects closed/private/user-cap/code-configured policy so it cannot bypass local registration controls.
- Remaining visibility risk is documentation/API wording drift around public challenge/account/score behavior and legacy `scoreboard_visible` naming.

Recommendation:

- Keep AstroCTFb's lifecycle model.
- Treat `visibility-registration-policy` as implemented for backend self-service registration controls.
- Continue cleanup on documentation drift around score/account/challenge route visibility and legacy `scoreboard_visible` naming.

Evidence: `A2`, `A9`, `A11`, `C1`, `C7`, `C10`.

### 3. Challenge and flag model

CTFd has a broader challenge model: challenge rows, separate `Flags`, static/regex flag classes, challenge topics, tags, files, hints, solutions, ratings, comments, and challenge logic such as `any`, `all`, and `team`.

AstroCTFb has a compact challenge table: one `flag_hash` plus optional encrypted regex flag, challenge-level and competition-level flag-format regex, state, dynamic scoring fields, files, tags, requirements, hints, solutions, ratings, and comments.

Difference:

- AstroCTFb is cleaner and easier to secure for the main submit path.
- CTFd supports richer challenge authoring and multi-flag semantics.

AstroCTFb plus:

- Constant-time hash comparison, encrypted regex flags, regex cache, semaphore, timeout, and global/challenge format regex are good backend security primitives.
- Compact schema helps avoid plugin-model complexity in v1.

AstroCTFb minus/risk:

- A single flag/hash-or-regex model is weaker for advanced challenge design.
- Missing `any/all/team` multi-flag semantics means no native "all team members must submit" or multi-stage partial capture model.
- AstroCTFb now has admin-only normalized topics separate from tags, but no player-facing topic discovery/filter API yet.

Recommendation:

- Do not import CTFd's plugin system now.
- Create a separate PRD for "challenge answer model v2" if advanced formats are in scope: multiple flags per challenge, flag groups, `any/all/team`, partial submissions, per-flag metadata, and migration strategy from current single-flag schema.

Evidence: `A7`, `A4`, `C3`, `C4`, `C5`.

### 4. Requirements, hints, solutions, unlocks

CTFd supports challenge prerequisites with reveal behavior such as anonymized or preview locked challenges. It also has polymorphic unlocks for hints and solutions, plus solution state.

AstroCTFb supports prerequisite edges, hints, hint unlocks, challenge files/writeups, one solution row per challenge, explicit solution states, and an admin unlock read model over existing hint unlocks.

Difference:

- AstroCTFb covers the main backend needs.
- CTFd is deeper in reveal/unlock semantics.

AstroCTFb plus:

- Requirement edges are normalized relational data, not only JSON-ish requirement blobs.
- Hint unlocks are per team and constrained for idempotency.
- Solution access is governed by explicit Astro-native states instead of an implicit leak-prone writeup path.
- `/admin/unlocks` has a generic unlock read shape while preserving the existing hint unlock write flow.

AstroCTFb minus/risk:

- Requirements do not yet appear to model CTFd-style `preview` / anonymized reveal.
- AstroCTFb intentionally does not add CTFd-like paid/free `SolutionUnlocks` writes in v1.
- Full polymorphic storage for non-hint unlock types remains deferred until there is a concrete product need.

Recommendation:

- Keep normalized requirement edges.
- Keep explicit solution visibility states as the v1 solution access policy.
- Keep the generic unlock read model backed by hint unlocks; only add new unlock write types when the product needs them.
- Decide whether challenge preview/anonymized reveal is important for international organizer workflows.

Evidence: `A7`, `C3`, `C4`.

### 5. Submission pipeline and abuse controls

CTFd's attempt endpoint is mature: it gates auth, event time, pause state, team-mode membership, hidden/locked challenge state, requirements, KPM anti-bruteforce, max attempts, statuses such as `correct`, `incorrect`, `partial`, `already_solved`, and `ratelimited`.

AstroCTFb's submit path is more high-load and consistency aware: it checks competition window, team membership, bans, competition mode, min team size, challenge visibility, requirements, max attempts, flag format, timing padding, regex ReDoS protection, atomic solve recording, cache invalidation, and realtime broadcast.

Difference:

- CTFd has broader product statuses, especially partials.
- AstroCTFb has a stronger concurrency and consistency story.

AstroCTFb plus:

- `singleflight` avoids DB stampedes on challenge fetch.
- Correct solves are rechecked in a transaction with competition/team/user/challenge locks.
- Advisory lock protects max-attempt accounting for team/challenge pairs.
- Regex matching is bounded by timeout and semaphore.

AstroCTFb minus/risk:

- No native partial-submission model equivalent to CTFd `Partials`.
- Submission status model lacks `partial`; current DB allows `correct`, `incorrect`, `ratelimited`, `discard`.

Recommendation:

- Keep the current submit pipeline as a core high-load advantage.
- If multi-flag challenge semantics are added, extend submission types and API responses deliberately rather than bolting partials into the current boolean result.

Evidence: `A4`, `A7`, `C4`, `C5`.

### 6. Scoring, freeze, tie-breaks, and scoreboard

CTFd computes standings by unioning solves and awards, filters banned/hidden accounts for public views, applies freeze, and resolves ties using achievement time and row ID. Dynamic scoring supports linear and logarithmic functions.

AstroCTFb supports dynamic score decay, stores `points_at_solve`, recalculates challenge points, records historical solve points, uses read-only transactions for scoreboard queries, and has local + Redis scoreboard caching with freeze-aware keys.

Current SQL evidence shows the public bracket scoreboard ordering as score descending, last solve time ascending with never-solved teams pushed to the end, then team ID ascending. Award values are included in the score sum and respect freeze time, but award timestamps do not appear to participate in the tie-break timestamp.

Difference:

- AstroCTFb is stronger on consistency and high-load cache design.
- CTFd's public tie-break behavior is more obvious and battle-tested.

AstroCTFb plus:

- `points_at_solve` is a strong design choice. It protects historical score correctness after later dynamic decay.
- Freeze-aware cache keys avoid mixing live and frozen views.
- Read-only transaction for scoreboard reads is a better high-load posture than ad hoc query paths.

AstroCTFb minus/risk:

- The exact public tie-break contract is visible in SQL but not yet documented as an organizer-facing rule.
- Award interaction is easy to misunderstand: awards affect score, but current evidence suggests they do not affect the last-solve tie-break timestamp.

Recommendation:

- Add a short scoring contract section to backend docs or this PRD: score sum, freeze cutoff, tie-break order, banned/hidden handling, bracket behavior, and award interaction.
- Add parity tests for CTFd-style ties: same score, different solve times, zero-point solves, awards before/after freeze, and award-only tie cases.

Evidence: `A5`, `A12`, `C6`, `C5`.

### 7. Users, teams, bans, brackets, custom fields

CTFd has mature user/team models with public/private hidden/banned behavior, brackets, invite codes, captain, supplementary profile fields, and custom fields with `description`, `required`, `public`, `editable`, and JSON values.

AstroCTFb has users, teams, brackets, captain, invite token, solo/auto-created teams, banned states, ban history references on solves/submissions/hints, user/team custom fields, and ban appeal usecases.

Difference:

- AstroCTFb is stronger in ban propagation and solo/team mechanics.
- CTFd is richer in user/team metadata and public field behavior.

AstroCTFb plus:

- `was_in_banned_team`, banned team/user references on activity, and ban appeal usecases are strong moderation primitives.
- Solo wrapper teams unify scoring without splitting the solve model into per-user vs per-team logic.

AstroCTFb minus/risk:

- Custom fields are improved for v1: schema now has `description`, `public`, `editable`, and typed API values; user and team profile flows enforce public/self visibility and editable patch rules while storage remains canonical string-backed.
- Profile metadata such as affiliation/country/website may be less complete than CTFd, depending on current API usage.

Recommendation:

- Keep solo wrapper teams.
- Keep custom field metadata and typed values Astro-native; do not clone CTFd field-entry APIs.
- Treat profile metadata as product scope, not high-load blocker.

Evidence: `A2`, `A3`, `A7`, `C3`, `C10`.

### 8. Auth, tokens, visibility, and rate limits

CTFd has email confirmation, reset password, registration visibility, registration code, password minimum length, user/team count caps, and ratelimited auth/team flows.

AstroCTFb has hashed API tokens with expirations and last-used tracking, Redis login lockout, route-specific dynamic rate-limit buckets, email verification/reset settings, OAuth toggles, and IP/user-agent tracking.

Difference:

- Both systems are mature enough for baseline auth.
- AstroCTFb looks stronger on API token storage and explicit route buckets.
- CTFd has more mature event-setup registration controls.

AstroCTFb plus:

- API tokens store only SHA-256 token hashes and validate plaintext by hashing at auth time.
- Route limit keys cover login, register, reset, OAuth, scoreboard, submit, hints, comments, ratings, profile updates, admin exports, destructive admin operations, file downloads, and WebSocket.
- Redis login lockout is explicit.

AstroCTFb minus/risk:

- AstroCTFb now has an Astro-native registration policy: optional `registration_code`, participant `max_users`, dynamic password minimum length, admin override, and OAuth new-account blocking when registration is closed/private/capped/code-gated.
- Separate account/challenge/score/registration visibility parameters exist, but their cross-effect with competition settings and public API documentation should continue to be documented.

Recommendation:

- Keep token and rate-limit architecture.
- Keep the product registration-policy comparison as a benchmark note only; do not claim or implement CTFd compatibility.

Evidence: `A9`, `A11`, `C7`, `C10`.

### 9. Admin API surface, content, import/export, statistics

CTFd's v1 API namespaces include challenges, tags, topics, awards, hints, flags, submissions, scoreboard, teams, users, statistics, files, notifications, configs, pages, unlocks, tokens, comments, shares, brackets, exports, and solutions.

AstroCTFb has OpenAPI routes for auth, user, team, challenge, submissions, scoreboard, statistics, competition, competition params, settings, fields, brackets, awards, tags, comments, notifications, pages, files, storage, backup, import/export/reset, API tokens, OAuth, email, avatar, setup, WebSocket/SSE.

Difference:

- CTFd has a wider mature product surface.
- AstroCTFb has strong source-of-truth API discipline and already covers many equivalent backend domains.
- AstroCTFb statistics are stronger than the previous first-pass wording implied, and now include an admin opened/attempted/solved progression funnel.

AstroCTFb plus:

- Backup/import/export/reset usecases are present and tested in current backend tree.
- OpenAPI route files make API coverage auditable.
- Statistics routes cover challenge solve counts, challenge detail, general counts, scoreboard history, solve percentages, score distribution, submission time series, registration series, an admin solve matrix, and an admin opened/attempted/solved funnel.
- Notifications cover public global history/counts, per-user inbox history/unread counts, admin personal delivery, and admin team fan-out delivery without adopting CTFd-compatible route semantics.
- Backup/import has version validation, ZIP bomb protection, async ZIP import job/status API, transactional database import, storage upload warning reporting, symlink skipping, storage path validation, metadata preflight for missing/rejected file payloads, optional SHA-256 file validation, a documented Astro-native archive contract, and a focused e2e runtime smoke.

AstroCTFb minus/risk:

- CTFd's polymorphic `unlocks` namespace is covered differently in AstroCTFb through an Astro-native admin unlock read model backed by hint unlocks; full non-hint unlock writes/storage are deferred. Shares are covered differently through Astro-native stateless signed solve links, and topics are covered differently through an Astro-native admin-only taxonomy.
- CTFd has mature raw CSV/ZIP export and long-lived import/export operations. AstroCTFb now has an Astro-native async ZIP import status model, documented archive contract, and focused runtime import smoke. CTFd-compatible migration is explicitly not a product goal.
- `challenge_opens` now deduplicates repeated page opens per `user_id + challenge_id`; public/admin statistics still use distinct/min semantics defensively, and retention cleanup reports deleted row counts.

Recommendation:

- Add a second-pass parity table for each CTFd namespace: equivalent, intentionally omitted, missing, or covered differently.
- Keep remaining product gaps explicit: full non-hint unlock writes/storage should only move forward if a concrete organizer workflow needs them after the implemented Astro-native data-portability, progression-funnel, notification, tracking, custom field metadata/typed values, team custom field profile flow, challenge metadata, challenge topic contracts, signed solve share links, and unlock read model.

Evidence: `A8`, `A13`, `A14`, `A15`, `A17`, `C8`, `C9`, `C11`, `C12`, `C13`.

### 10. API contracts, architecture, realtime, testing, and high-load readiness

AstroCTFb follows Clean Architecture boundaries, documents handler rules, uses OpenAPI as the source of truth, bundles/generates OpenAPI types/server/spec/client, verifies sqlc, runs mockery/wire, and defines test targets for unit, integration, e2e, race, load, load reports, and benchmarks.

CTFd is a mature Flask application with a wide model/API surface and broad regression tests, but the local evidence shows a more coupled model/config/plugin style and less explicit high-load contract tooling.

Difference:

- AstroCTFb is stronger as an engineered backend for high-load operations.
- CTFd is stronger as a long-lived, community-hardened CTF product baseline.

AstroCTFb plus:

- OpenAPI-first development is a major maintainability advantage.
- Backend realtime is explicit: `/ws`, SSE/event envelope, async broadcaster, and WebSocket endurance/load tests.
- Race/load targets and hot-path benchmarks match the stated "international highload" positioning.

AstroCTFb minus/risk:

- Strong backend engineering does not automatically close product-surface gaps.
- Since frontend/plugins/per-instance customization are excluded here, this document cannot claim whole-product superiority over CTFd.

Recommendation:

- Position AstroCTFb as "high-load CTF backend/core with stricter contracts and operations" rather than "CTFd clone".
- Keep using CTFd as product capability checklist, not as architecture template.

Evidence: `A1`, `A6`, `A10`, `C8`.

## Prioritized decisions

## Implementation progress

| Target | Status | Notes |
| --- | --- | --- |
| `remove-flexible-mode` | done | `flexible` removed from backend domain/config/setup/update validation, DB constraints, OpenAPI/generated schemas, minimal frontend mode branches, README, and stale tracked report artifact. Remaining mentions are negative tests only. |
| `scoring-contract-and-fairness` | done | Backend scoring contract is documented, `/scoreboard` OpenAPI wording states ordering/filtering/freeze behavior, and integration tests pin tie-break, zero-point solve, award freeze, and award-only tie semantics. |
| `challenge-answer-model-v2` | done | Current single hash/regex answer model is accepted for v1. Multi-flag groups, `any/all/team`, partial submissions, and migration strategy are deferred to a separate v2 PRD without runtime/schema changes in this release. |
| `solution-unlock-policy` | done | Astro-native v1 adds explicit solution states (`hidden`, `solved_only`, `after_event`, `admin_only`) in the squashed base schema and applies them to solution content and writeup files. Admin saves without `state` preserve current state, and CTFd-compatible `SolutionUnlocks`/paid solution unlocks remain out of scope. |
| `statistics-funnel` | done | Admin statistics now expose opened/attempted/solved progression for challenges and top teams/users through an Astro-native endpoint. SQL and tests cover untouched, opened-only, attempted-only, solved, hidden/banned filtering, freeze/live behavior, bounded queries, response mapping, and challenge-open cache invalidation. |
| `visibility-registration-policy` | done | Registration visibility/code/user-cap/password-policy is enforced across local registration, OAuth new-account creation, admin overrides, OpenAPI/settings surfaces, sqlc, and the squashed base migration set. |
| `data-portability` | done | Astro-native backup/import now has async ZIP import jobs, documented ZIP/JSON archive contract, OpenAPI `BackupData` alignment, safer file restore preflight, focused unit coverage, and filesystem-storage e2e import smoke. |
| `notifications-tracking-parity` | done | Astro-native notifications now expose global count with timestamp cursor, current-user unread count, admin personal delivery, admin team fan-out delivery to current non-banned team members, transactional fan-out, generated OpenAPI/client/server contracts, and focused usecase coverage. Tracking cleanup now returns deleted row counts for tracking and challenge-open records; challenge-open writes are deduplicated per user/challenge through the squashed base index migration. |
| `custom-field-richness` | done | Custom field definitions now have `description`, `public`, and `editable`; own profile responses include own values, public profiles expose only public values, and profile patch supports partial editable user field updates. |
| `challenge-metadata-envelope-v1` | done | Challenge contracts now carry `attribution` and nullable `next_id`; DB, sqlc, OpenAPI, REST mapping, backup JSON/CSV, repository round-trip, and locked-response redaction paths are covered. |
| `challenge-topics-v1` | done | Astro-native admin-only topic taxonomy now has normalized `topics`, challenge-topic assignments, admin REST/OpenAPI routes, backup JSON import/export, focused usecase/mapper tests, integration repository round-trip, and no public player challenge exposure. |
| `share-links-v1` | done | Astro-native signed solve share links are implemented with verified-user creation, public MAC validation, escaped HTML/OG preview, and visibility/ban/soft-delete guards. |
| `custom-field-json-values` | done | Custom fields now expose typed API values while keeping canonical string storage for field values and Astro-native backups. |
| `unlocks-v1` | done | `/admin/unlocks` now returns a generic Astro-native unlock read model over existing hint unlocks. Hint unlock writes remain unchanged, and solution unlock charging stays deferred. |

### Must decide before international high-load positioning

1. Remove `flexible` mode:
   - keep only `teams_only` and `solo_only`;
   - remove mixed-mode behavior from domain, DB constraints, OpenAPI, setup/admin UI, frontend schema, guards, and tests.
2. Define scoring and fairness contract:
   - `solo_only`: individual scoreboard/awards.
   - `teams_only`: team scoreboard/awards, recommended default for serious team CTF.
3. Document scoring mechanics:
   - score source, `points_at_solve`, awards, freeze cutoff, tie-break order, bracket scope, and hidden/banned filtering.
4. Decide challenge answer model v2:
   - keep single flag for v1, or add multi-flag groups, `any/all/team`, partial submissions, and migration from the current schema.
5. Decide solution/unlock semantics:
   - hidden, visible, solved-only, paid unlock, free unlock, public-after-event, duplicate unlock behavior, and API response shape.
6. Maintain the CTFd namespace parity table:
   - equivalent / covered differently / missing / intentionally omitted / out-of-scope.

### Should decide next

- No active backend/core item is selected. Challenge-open insert dedupe, custom field metadata/editability, team custom field self-edit/profile flow, challenge metadata envelope v1, challenge topics v1, signed solve share links, custom field JSON values, and unlocks v1 are implemented. Full non-hint unlock writes remain a future product PRD only if needed.

### Could decide later

- Paid/free solution unlock writes or full non-hint unlock storage, if a concrete organizer workflow needs them later.

### Good divergence from CTFd

- Keep Clean Architecture and OpenAPI-first contracts.
- Keep Go/PostgreSQL/Redis-first high-load backend posture.
- Keep solo wrapper teams for unified scoring.
- Keep transaction rechecks, row/advisory locks, regex timeout/semaphore, and cache invalidation discipline.
- Keep WebSocket/SSE realtime as a first-class backend feature.

### Accepted gaps for now

- Multi-DB support: PostgreSQL-first is acceptable for high-load if migrations and operations are strong.
- Plugin ecosystem: explicitly out of scope for this comparison.
- Themes/frontend customization: explicitly out of scope.
- Per-instance/deployment customization: explicitly out of scope.
- Social/share features: lower priority than scoring, challenge model, statistics, and portability.

## Evidence appendix

AstroCTFb:

- `A1`: `docs/ARCHITECTURE.md:24-40`, `docs/ARCHITECTURE.md:110-118`, `CONTRIBUTING.md:136-145` - backend boundaries, handler rules, OpenAPI source of truth.
- `A2`: `backend/internal/domain/competition.go:7-24`, `backend/internal/domain/competition.go:43-76`, `backend/migrations/000001_init.sql:10-35` - competition lifecycle and mode schema.
- `A3`: `backend/internal/usecase/team/team_solo.go:16-100`, `backend/internal/usecase/guard/eligibility.go:28-76` - solo team creation and submission eligibility.
- `A4`: `backend/internal/usecase/challenge/challenge_submit.go:39-53`, `backend/internal/usecase/challenge/challenge_submit.go:137-204`, `backend/internal/usecase/challenge/challenge_submit_record.go:17-190`, `backend/internal/usecase/challenge/challenge_submit_flag.go:16-170` - submit path, locks, max attempts, regex safety, flag checks.
- `A5`: `backend/internal/scoring/scoring.go:13-74`, `backend/internal/scoring/scoring.go:76-144`, `backend/internal/usecase/competition/solve_record.go:14-72`, `backend/internal/usecase/competition/solve.go:245-335` - dynamic scoring, points at solve, freeze-aware scoreboard cache.
- `A6`: `backend/internal/openapi/routes/websocket.yml:1-90`, `backend/internal/websocket/broadcaster.go:12-120`, `backend/load-test/websocket_test.go:18-80` - realtime contract, broadcaster, WebSocket load/endurance tests.
- `A7`: `backend/migrations/000001_init.sql:184-210`, `backend/migrations/000001_init.sql:228-287`, `backend/migrations/000001_init.sql:294-328`, `backend/migrations/000001_init.sql:480-502` - challenge, requirements, hints, solutions, solves, submissions, fields.
- `A8`: `backend/internal/openapi/routes/`, `backend/internal/usecase/` - route/usecase surface inspected by file inventory.
- `A9`: `backend/internal/controller/restapi/v1/router_limits.go:15-109`, `backend/internal/usecase/user/apitoken.go:46-128`, `backend/internal/loginlockout/loginlockout.go:14-109`, `backend/migrations/000001_init.sql:37-72`, `backend/migrations/000001_init.sql:522-546` - rate limits, API tokens, login lockout, settings, tracking.
- `A10`: `backend/Makefile:43-49`, `backend/Makefile:90-141`, `backend/Makefile:234-239`, `backend/Makefile:262-356` - pinned tools, OpenAPI/sqlc/codegen checks, test/race/load/bench gates.
- `A11`: `backend/internal/domain/config_registry.go:65-80`, `backend/internal/usecase/competition/competition_params.go:54-59` - separate challenge, score, account, and registration visibility parameters and allowed values.
- `A12`: `backend/queries/solves.sql:61-86` - bracket scoreboard score sum, award inclusion, freeze filter, and tie-break ordering.
- `A13`: `backend/internal/openapi/routes/statistics.yml:1-489`, `backend/queries/statistics.sql:72-90`, `backend/queries/statistics.sql:138-229` - statistics routes, solve percentages, score/submission/registration series, and solve matrix.
- `A14`: `backend/internal/usecase/user/tracking.go:28-72`, `backend/queries/challenge_opens.sql:1-13` - tracking entries and challenge-open events.
- `A15`: `backend/internal/usecase/backup/backup_export.go:20-115`, `backend/internal/usecase/backup/backup_import.go:14-100`, `backend/internal/usecase/backup/backup_import_job.go:23-193`, `backend/internal/usecase/backup/backup_import_files.go:23-235`, `backend/internal/openapi/routes/backup.yml:276-386` - versioned backup/export/import, ZIP limits, async import status, transactional import, partial file upload reporting, symlink skip, path validation, file metadata preflight, and hash validation.
- `A16`: `backend/internal/usecase/notification/notification.go`, `backend/internal/controller/restapi/v1/notification.go`, `backend/internal/openapi/routes/notification.yml`, `backend/queries/notifications.sql`, `backend/internal/usecase/cleanup/cleanup.go`, `backend/queries/tracking.sql` - Astro-native global/user/team notification contracts, notification counts, team fan-out, and tracking cleanup deleted-row counts.
- `A17`: `backend/internal/usecase/competition/share.go`, `backend/internal/controller/restapi/v1/share.go`, `backend/internal/openapi/routes/share.yml`, `backend/internal/openapi/components/schemas/share_schemas.yml`, `backend/internal/domain/config_registry.go`, `backend/config/build.go` - Astro-native signed solve share links, public HTML preview contract, dynamic share disable key, and separate derived share signing secret.

CTFd:

- `C1`: `CTFd/CTFd/constants/options.py:4-43` - config/user-mode/visibility enum values.
- `C2`: `CTFd/CTFd/utils/modes/__init__.py:7-41` - `users` / `teams` mode helper behavior.
- `C3`: `CTFd/CTFd/models/__init__.py:273-390`, `CTFd/CTFd/models/__init__.py:393-590`, `CTFd/CTFd/models/__init__.py:619-730`, `CTFd/CTFd/models/__init__.py:930-1004`, `CTFd/CTFd/models/__init__.py:1084-1163` - topics, solutions, files, flags, users, teams, submissions, unlocks, comments, fields.
- `C4`: `CTFd/CTFd/api/v1/challenges.py:180-240`, `CTFd/CTFd/api/v1/challenges.py:646-900` - requirements reveal and attempt pipeline.
- `C5`: `CTFd/CTFd/plugins/challenges/logic.py:7-130`, `CTFd/CTFd/plugins/challenges/decay.py:9-75` - multi-flag logic and dynamic scoring functions. Plugin loading is excluded; these behaviors are still CTFd challenge capabilities.
- `C6`: `CTFd/CTFd/utils/scores/__init__.py:10-134`, `CTFd/CTFd/utils/scores/__init__.py:137-180` - standings, freeze, banned/hidden filtering, tie-break ordering.
- `C7`: `CTFd/CTFd/utils/config/visibility.py:12-51` - challenge/score/account/registration visibility helpers.
- `C8`: `CTFd/CTFd/api/__init__.py:54-75` - API namespace surface.
- `C9`: `CTFd/CTFd/api/v1/statistics/challenges.py:13-120`, `CTFd/CTFd/api/v1/statistics/progression.py:12-120` - challenge statistics and progression matrix.
- `C10`: `CTFd/CTFd/teams.py:56-121`, `CTFd/CTFd/teams.py:124-230`, `CTFd/CTFd/auth.py:37-90`, `CTFd/CTFd/auth.py:236-285` - team invite/join/create and auth/registration controls.
- `C11`: `CTFd/CTFd/api/v1/config.py:40-125`, `CTFd/CTFd/api/v1/config.py:203-286` - runtime config and `/configs/fields` API surface with cache invalidation.
- `C12`: `CTFd/CTFd/api/v1/challenges.py:575-590`, `CTFd/CTFd/api/v1/statistics/progression.py:12-203` - challenge-open tracking and opened/attempted/solved progression matrix.
- `C13`: `CTFd/CTFd/api/v1/exports.py:14-46`, `CTFd/CTFd/utils/exports/__init__.py:41-94`, `CTFd/CTFd/utils/exports/__init__.py:448-492`, `CTFd/CTFd/utils/exports/__init__.py:507-523` - raw/ZIP export, upload restoration, path traversal checks, cache clearing, and background import lifecycle.
- `C14`: `CTFd/CTFd/api/v1/notifications.py:40-153`, `CTFd/CTFd/events/__init__.py:10-26`, `CTFd/CTFd/models/__init__.py:1007-1024` - persisted notifications, SSE events, and tracking model.
