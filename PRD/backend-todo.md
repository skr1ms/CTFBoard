# AstroCTFb backend TODO

Status: draft, living local TODO
Date: 2026-06-07
Source: `PRD/analyze.md`

Scope: backend/core product behavior, API contracts, persistence, high-load readiness, and tests.

Non-goals:

- Do not implement CTFd-compatible import/export. CTFd is a benchmark and competitor, not a compatibility target.
- Do not work on frontend UX polish, themes, plugin ecosystem, or per-instance customization in this TODO.
- Do not rewrite architecture to look like CTFd. Keep Go, PostgreSQL, Redis, Clean Architecture, and OpenAPI-first contracts.

## Implementation progress

| Target | Status | Notes |
| --- | --- | --- |
| `remove-flexible-mode` | done | `flexible` is removed from runtime validation, DB/OpenAPI constraints, setup/update flows, generated schemas, and stale UI/docs references. Remaining mentions are negative tests or historical analysis. |
| `scoring-contract-and-fairness` | done | Scoring/freeze/tie-break behavior is documented and covered by contract tests. |
| `challenge-answer-model-v2` | done | v1 keeps the current single hash/regex answer model; multi-flag groups, `any/all/team`, partial submissions, and migration strategy are deferred to `PRD/challenge-answer-model-v2.md`. |
| `solution-unlock-policy` | done | Astro-native solution states gate solution content and writeup files; CTFd-compatible solution unlock resources remain out of scope. |
| `statistics-funnel` | done | Admin opened/attempted/solved funnel is implemented and verified across SQL, response mapping, freeze/live behavior, filtering, bounded queries, and challenge-open cache invalidation. |
| `visibility-registration-policy` | done | Registration visibility/code/user-cap/password-policy contract is implemented in backend usecases, OAuth new-account flow, OpenAPI/settings surfaces, sqlc, and the squashed base migration set. |
| `data-portability` | done | Astro-native ZIP/JSON archive contract is documented, OpenAPI `BackupData` matches exported fields, file restore preflight has focused tests, and async ZIP import has a filesystem-storage e2e smoke. |
| `notifications-tracking-parity` | done | Astro-native notification count/unread/team fan-out contracts are implemented; tracking cleanup reports deleted row counts, and challenge-open writes are deduplicated by user/challenge. |
| `custom-field-richness` | done | Custom fields now carry `description`, `public`, and `editable`; `/auth/me` exposes own values, public profiles expose only public values, and profile update supports partial editable user field updates. |
| `team-custom-field-self-edit-profile` | done | Team custom fields now have public/self visibility and captain-only partial editable updates through the team profile flow. |
| `challenge-metadata-envelope-v1` | done | Challenge contracts now carry `attribution` and optional `next_id`; DB, sqlc, OpenAPI, REST mapping, backup JSON/CSV, and repo round-trip paths are covered. |
| `challenge-topics-v1` | done | Astro-native admin-only topic taxonomy is implemented with normalized topics, challenge assignments, OpenAPI/admin routes, backup JSON round-trip, and focused unit/integration coverage. |
| `share-links-v1` | done | Astro-native signed solve share links are implemented with verified-user creation, public MAC validation, OG/HTML preview, and visibility/ban/soft-delete guards. |
| `custom-field-json-values` | done | Custom field values are now typed at the API boundary: text/select strings, integral numbers, booleans, and arbitrary JSON values, while storage stays canonical string-backed. |
| `unlocks-v1` | done | `/admin/unlocks` now exposes an Astro-native unlock read model over existing hint unlocks; solution unlock charging remains deferred. |

## Priority map

| Status | Priority | Task | Why it matters |
| --- | --- | --- | --- |
| Done | P0 | Remove `flexible` mode | It removes mixed solo/team complexity without a clear event use case. |
| Done | P1 | Scoring and fairness contract | International events need predictable scoreboard and award rules. |
| Done | P1 | Challenge answer model v2 decision | The v1 contract is explicit; advanced answer models are deferred to a separate PRD. |
| Done | P1 | Solution and unlock policy | Writeups/solutions now have explicit visibility and unlock semantics before player exposure. |
| Done | P1 | Admin statistics funnel | Admins need opened/attempted/solved progression, not only solves. |
| Done | P2 | Visibility and registration policy | Registration controls are enforced without making the platform CTFd-compatible. |
| Done | P2 | Astro-native data portability | Backup/import exists with documented Astro-native archive contract, async status, safer file restore, and runtime e2e smoke. |
| Done | P2 | Notifications and tracking parity | Realtime remains strong, and persisted notification/tracking behavior now has clearer Astro-native contracts. |
| Done | P3 | Custom field richness | Organizer registration/profile fields now have description/privacy/editability metadata and value visibility rules. |
| Done | P3 | Team custom field self-edit/profile flow | Teams can expose public custom fields and captains can patch editable team fields without changing roster/name policy. |
| Done | P3 | Challenge metadata envelope v1 | Organizer-facing challenge metadata now includes attribution and next-challenge navigation without adding CTFd-compatible challenge logic. |
| Done | P3 | Challenge topics v1 | Organizers can manage normalized challenge taxonomy separately from tags without exposing a player-facing topic API/filter yet. |
| Done | P3 | Solve share links v1 | Participants can share signed public solve previews without adding CTFd-compatible share APIs. |
| Done | P3 | Custom field JSON values | Organizer custom fields can now carry typed JSON values without changing the field-value storage table. |
| Done | P3 | Unlocks v1 | Admins can inspect unlock records through a generic Astro-native unlock contract without adding CTFd-compatible solution unlock writes. |

## Latest implementation snapshot

Last backend TODO completed: `unlocks-v1`.
Current backend TODO in progress: none selected.

Verification completed:

- Latest `unlocks-v1` pass: `make openapi`
- Latest `unlocks-v1` pass: `make validate-openapi`
- Latest `unlocks-v1` pass: `make sqlc-verify`
- Latest `unlocks-v1` pass: `go test ./internal/controller/restapi/v1/response ./internal/usecase/challenge ./internal/controller/restapi/v1 -count=1`
- Latest `unlocks-v1` pass: `go test ./integration-test -run '^(TestHintUnlockRepo_Flow|TestHintUnlockRepo_Rollback_UnlockNotPersistedOnError|TestHintUnlockRepo_Rollback_AwardOrphan|TestHintUnlockAndAwardTx_Commit|TestHintUnlock_SequentialCorrect|TestHintRepo_SoftBanUnlocksByTeamID|TestHintRepo_RestoreUnlocksByBannedTeamID)$' -count=1 -v`
- Latest `unlocks-v1` pass: `go test ./integration-test -run '^TestHintUnlock_ConcurrentBalanceRace$' -count=1 -v`
- Latest `unlocks-v1` pass: `go test ./e2e-test -run '^TestE2E_HintAndFileJourney$' -count=1 -v`
- Latest `unlocks-v1` pass: `make lint`
- Latest `unlocks-v1` pass: `git diff --check`
- Latest `custom-field-json-values` pass: `make openapi`
- Latest `custom-field-json-values` pass: `make validate-openapi`
- Latest `custom-field-json-values` pass: `go test ./internal/usecase/settings ./internal/usecase/user ./internal/usecase/team ./internal/controller/restapi/v1/request ./internal/controller/restapi/v1/response ./pkg/validator -count=1`
- Latest `custom-field-json-values` pass: `go test ./internal/usecase/... -count=1`
- Latest `custom-field-json-values` pass: `go test ./internal/controller/restapi/... -count=1`
- Latest `custom-field-json-values` pass: `go test ./integration-test -run '^(TestFieldRepo_Create_JSONField|TestBackupRepo_ImportFieldsTx_Success)$' -count=1 -v`
- Latest `challenge-topics-v1` pass: `make sqlc`
- Latest `challenge-topics-v1` pass: `make openapi`
- Latest `challenge-topics-v1` pass: `make generate-mocks-challenge generate-mocks-backup`
- Latest `challenge-topics-v1` pass: `make wire`
- Latest `challenge-topics-v1` pass: `make sqlc-verify`
- Latest `challenge-topics-v1` pass: `make validate-openapi`
- Latest `challenge-topics-v1` pass: `go test ./internal/usecase/challenge ./internal/usecase/backup ./internal/controller/restapi/v1/request ./internal/controller/restapi/v1/response ./internal/controller/restapi/v1 ./internal/wire`
- Latest `challenge-topics-v1` pass: `go test ./integration-test -run '^TestTopicRepo_' -count=1 -v`
- Latest `challenge-topics-v1` pass: `go test ./integration-test -run '^(TestBackupRepo_ImportTopicsAndChallengeTopicsTx_Success|TestBackupUseCase_ExportIncludesTopics)$' -count=1 -v`
- Latest `challenge-topics-v1` pass: `make test-fast`
- Latest `challenge-topics-v1` pass: `make lint`
- Latest `challenge-topics-v1` pass: `make audit-architecture-strict`
- Latest `challenge-topics-v1` pass: `git diff --check`
- Latest `team-custom-field-self-edit-profile` pass: `make openapi wire`
- Latest `team-custom-field-self-edit-profile` pass: `go test ./internal/usecase/team ./internal/controller/restapi/v1/request`
- Latest `team-custom-field-self-edit-profile` pass: `go test ./internal/controller/restapi/v1`
- Latest `team-custom-field-self-edit-profile` pass: `go test ./internal/wire`
- Latest `team-custom-field-self-edit-profile` pass: `make validate-openapi`
- Latest `team-custom-field-self-edit-profile` pass: `go test ./internal/usecase/...`
- Latest `team-custom-field-self-edit-profile` pass: `go test ./internal/controller/restapi/...`
- Latest `team-custom-field-self-edit-profile` pass: `go test ./e2e-test -run '^$'`
- Latest `team-custom-field-self-edit-profile` pass: `make lint`
- Latest `team-custom-field-self-edit-profile` pass: `git diff --check`
- Latest `challenge-metadata-envelope-v1` pass: `make sqlc`
- Latest `challenge-metadata-envelope-v1` pass: `make openapi`
- Latest `challenge-metadata-envelope-v1` pass: `go test ./internal/controller/restapi/v1/request ./internal/controller/restapi/v1/response ./internal/usecase/challenge ./internal/usecase/backup`
- Latest `challenge-metadata-envelope-v1` pass: `go test ./integration-test -run '^(TestChallengeRepo_MetadataRoundTrip|TestBackupRepo_ImportChallengesTx_Success)$' -count=1 -v`
- Latest `challenge-metadata-envelope-v1` pass: `make sqlc-verify`
- Latest `challenge-metadata-envelope-v1` pass: `make validate-openapi`
- Latest `challenge-metadata-envelope-v1` pass: `make test-fast`
- Latest `challenge-metadata-envelope-v1` pass: `make lint`
- Latest `challenge-metadata-envelope-v1` pass: `git diff --check`
- Latest `custom-field-richness` pass: `make sqlc`
- Latest `custom-field-richness` pass: `make openapi`
- Latest `custom-field-richness` pass: `make wire`
- Latest `custom-field-richness` pass: `go test ./internal/usecase/settings ./internal/usecase/user ./internal/controller/restapi/v1/request ./internal/controller/restapi/v1/response ./internal/usecase/backup`
- Latest `custom-field-richness` pass: `go test ./internal/controller/restapi/v1`
- Latest `custom-field-richness` pass: `go test ./integration-test -run '^(TestFieldRepo_.*|TestFieldValueRepo_.*|TestBackupRepo_ImportFieldsTx_Success)$' -count=1 -v`
- Latest `custom-field-richness` pass: `make sqlc-verify`
- Latest `custom-field-richness` pass: `make validate-openapi`
- Latest `custom-field-richness` pass: `make test-fast`
- Latest `custom-field-richness` pass: `make lint`
- Latest `custom-field-richness` pass: `git diff --check`
- Latest `challenge-open-write-dedupe` pass: `make sqlc`
- Latest `challenge-open-write-dedupe` pass: `go test ./integration-test -run '^TestTrackingRepo_CreateChallengeOpen_DeduplicatesByUserAndChallenge$' -count=1 -v`
- Latest `challenge-open-write-dedupe` pass: `go test ./integration-test -run '^(TestTrackingRepo_CreateChallengeOpen_DeduplicatesByUserAndChallenge|TestStatisticsRepo_GetAdminStatisticsFunnel)' -count=1 -v`
- Latest `challenge-open-write-dedupe` pass: `go test ./internal/usecase/user -run '^TestTrackingUseCase' -count=1`
- Latest `challenge-open-write-dedupe` pass: `make test-fast`
- Latest `challenge-open-write-dedupe` pass: `make lint`
- Latest `challenge-open-write-dedupe` pass: `git diff --check`
- Latest migration-squash adjustment: `go test ./integration-test -run '^TestTrackingRepo_CreateChallengeOpen_DeduplicatesByUserAndChallenge$' -count=1 -v`
- Latest migration-squash adjustment: `make sqlc`
- Latest migration-squash adjustment: `make test-fast`
- Latest migration-squash adjustment: `make lint`
- Latest migration-squash adjustment: `git diff --check`
- `make openapi`
- `make validate-openapi`
- `go test ./internal/usecase/backup`
- `go test ./internal/controller/restapi/v1/request`
- `go test -parallel 1 -v ./e2e-test/... -run '^TestE2E_BackupImportJobSmoke$' -count=1`
- `make lint`
- `git diff --check`
- `make sqlc`
- `make openapi`
- `make wire`
- `make generate-mocks-notification generate-mocks-user generate-mocks-middleware`
- `go test ./internal/usecase/notification ./internal/usecase/cleanup ./internal/repo/persistent ./internal/usecase/user ./internal/controller/restapi/middleware ./internal/controller/restapi/v1 ./cmd/cleanup`
- `make test-fast`

Skipped checks:

- Full `make test-e2e`, load, and S3/SeaweedFS runtime restore smoke were not run in this pass. The focused import smoke used the existing e2e testcontainers PostgreSQL/Redis harness and filesystem storage.

Worktree note:

- The repository still has a large dirty tree from multiple completed TODO slices. Commit next by semantic areas instead of one mixed commit.

## P0: remove-flexible-mode

Analyze reference: `PRD/analyze.md` -> `Decision register`, `Participation model`, `Prioritized decisions`.

Implementation status: done. `flexible` is removed from backend domain/config/setup/update
validation, DB constraints, OpenAPI/generated schemas, minimal frontend mode branches,
README, and stale tracked report artifacts. Remaining `flexible` mentions are negative
tests or historical analysis, not accepted runtime behavior.

Result:

- Only `teams_only` and `solo_only` are valid competition modes.
- `teams_only` remains default.
- Solo wrapper teams remain for `solo_only`, because they keep scoring unified.

## P1: scoring-contract

Analyze reference: `PRD/analyze.md` -> `Scoring, freeze, tie-breaks, and scoreboard`.

Implementation status: done. The scoring contract is written and enforced by tests.
It covers score sources, `points_at_solve`, awards, freeze cutoff, tie-break order,
bracket scope, and hidden/banned filtering.

- Tests cover same score with different solve times.
- Tests cover award before/after freeze.
- Tests cover award-only teams.
- Tests cover hidden/banned teams not appearing in public scoreboard.

## P1: challenge-answer-model-v2

Analyze reference: `PRD/analyze.md` -> `Challenge and flag model`, `Submission pipeline and abuse controls`.

Implementation status: done as a product/technical decision for v1. AstroCTFb keeps
the current single `flag_hash` or encrypted regex flag model for this release.
Advanced challenge authoring belongs to `PRD/challenge-answer-model-v2.md`, not the
current backend TODO.

Deferred v2 scope:

- Multiple flags per challenge.
- `any/all/team` solve logic.
- `partial` submission state.
- Migration path from existing single-flag challenges.

## P1: solution-unlock-policy

Analyze reference: `PRD/analyze.md` -> `Requirements, hints, solutions, unlocks`.

Implementation status: done for Astro-native v1. The backend now uses explicit solution
visibility states and applies them to solution content plus writeup files. Separate
CTFd-compatible `SolutionUnlocks` and paid solution unlock charging are intentionally
deferred. The squashed base schema stores `solutions.state`, and legacy admin solution
saves that omit `state` preserve the current state instead of resetting hidden/admin-only
solutions to solved-only.

Resolved issue:

- AstroCTFb had solution/writeup data without a clear player visibility contract.
- CTFd has richer hidden/visible/solved/unlocked semantics.
- AstroCTFb now covers the backend-safe v1 path with explicit solution states; paid/free
  solution unlock writes are still intentionally deferred.

Implementation surfaces:

- Solution schema and queries.
- Challenge detail response.
- Hint/solution unlock flows.
- Admin solution management API.

Expected result:

- Solutions have explicit states such as hidden, visible after solve, visible after event, and admin-only.
- Unlock behavior is not added for solutions in v1; paid/free solution unlocks are deferred.
- Player API response never leaks hidden solution content.

Acceptance criteria:

- Tests cover unsolved player, solved player, admin, and after-event views.
- Existing DBs get a non-destructive migration for `solutions.state`.
- Admin upsert without a `state` field preserves an existing solution state.
- Duplicate/paid solution unlock handling is deferred because solution unlock records are not part of v1.
- Hidden solution content is not returned by public challenge APIs.

## P1: statistics-funnel

Analyze reference: `PRD/analyze.md` -> `Admin API surface, content, import/export, statistics`.

Implementation status: done. AstroCTFb now has an Astro-native admin funnel for
opened, attempted, and solved progression across challenges and top teams/users.
The endpoint is bounded by limit, uses the short statistics cache, respects frozen
scoreboard mode unless admin requests live data, and invalidates funnel cache when
challenge-open tracking records are written.

Original issue:

- Current statistics cover solves, solve percentages, score distribution, time series, registration series, and solve matrix.
- Missing piece is a CTFd-like admin funnel that distinguishes opened, attempted, and solved.
- AstroCTFb already tracks challenge opens, but current evidence does not show a combined opened/attempted/solved matrix response.

Implementation surfaces:

- `challenge_opens` tracking.
- Submissions/solves statistics queries.
- Admin statistics OpenAPI routes.
- Cache/freeze/live admin behavior.

Expected result:

- Admin can see per challenge and per top team/user: opened, attempted, solved.
- Funnel respects hidden/banned filtering and freeze/live admin mode.
- Heavy matrix queries are cached or bounded.

Acceptance criteria:

- Done: tests cover opened without attempt, attempted without solve, solved, and never opened.
- Done: tests cover hidden/banned teams/users exclusion.
- Done: tests cover freeze behavior; usecase tests cover frozen cache path vs admin live request.
- Done: query limits and short-cache behavior prevent unbounded matrix cost.
- Done: response mapping and challenge-open cache invalidation are covered.

## P2: visibility-registration-policy

Analyze reference: `PRD/analyze.md` -> `Competition lifecycle and visibility`, `Auth, tokens, visibility, and rate limits`.

Implementation status: done. Because there is no production DB yet, the additive
registration migration was squashed into the base migration set: `app_settings.max_users`
is in `000001_init.sql`, and `account_visibility` is seeded by `000003_seed.sql`.

Current issue:

- Separate visibility params exist: challenge, score, account, registration.
- Registration controls now follow an explicit backend contract: `registration_open=false` closes self-service registration, `registration_visibility=private` hides/closes self-service registration, optional `registration_code` gates password registration, and `max_users` caps participant accounts.
- Admin-created users intentionally bypass registration visibility/code/user-cap controls, while password policy still applies.
- OAuth new-account registration no longer bypasses closure/private/user-cap/code-configured policy; existing OAuth login/link flows stay available.

Implementation surfaces:

- Competition params/settings.
- Auth/register flow.
- OAuth new-account registration flow.
- OpenAPI admin settings/auth schemas.
- User repository/sqlc/migrations.

Expected result:

- One registration/visibility contract exists and is enforced.
- Admin can configure registration closure, optional registration code, participant user cap, team cap, and relevant password policy values.
- Existing `max_teams` remains enforced through team creation and solo-team auto-create paths.

Acceptance criteria:

- Done: tests cover registration private/closed behavior and public successful registration.
- Done: tests cover valid, missing, invalid, and case-insensitive registration code.
- Done: tests cover user cap reached for local registration and OAuth new-account registration; existing team-cap tests still cover team creation.
- Done: tests cover admin user creation override where intended.
- Done: tests cover dynamic password minimum length in registration and reset flows.
- Done: base schema includes `app_settings.max_users`, and base seed includes `account_visibility`.
- Done: OpenAPI, sqlc, mocks, wire, lint, and targeted/full backend package tests pass.

Residual notes:

- OAuth with configured `registration_code` blocks new account creation instead of offering a code-entry OAuth UX. This is intentional for now because it prevents bypass.
- `max_users` counts active, non-banned participant accounts (`role=user AND is_banned=false`). If banned users must still consume seats, this contract needs a small follow-up change.
- A live migration smoke (`goose up`) should be run when PostgreSQL/runtime is available.

## P2: data-portability

Analyze reference: `PRD/analyze.md` -> `Data portability`, `Admin API surface, content, import/export, statistics`.

Implementation status: done. AstroCTFb now has async ZIP import jobs,
operator-visible job status, stricter file restore preflight, a documented
Astro-native archive contract, OpenAPI schema alignment for exported backup
fields, and a focused filesystem-storage e2e import smoke.

Current issue:

- AstroCTFb backup/import is already strong, but needs a stable product contract.
- CTFd-compatible import/export is not required and should not be planned.
- Operators need predictable archive versioning, import status, restore order, file validation, and partial failure behavior.

Implementation surfaces:

- Backup/export/import usecases.
- Backup archive schema/version.
- Import result/status API with async ZIP import job persistence, `202 Accepted` job creation, and `GET /admin/import/jobs/{ID}`.
- Storage file validation.
- Admin reset/import/export routes.

Expected result:

- Astro-native export/import format is documented and versioned.
- Import exposes clear status/errors through queued/running/completed/failed status, coarse phase, result, and failure text.
- File restore behavior is explicit for missing files, hash mismatch, path validation, symlinks, and partial upload failures. Invalid file metadata is skipped before DB import, so metadata no longer points to missing or rejected payloads.

Acceptance criteria:

- Done: tests cover backup version mismatch.
- Done: tests cover missing `backup.json`.
- Done: tests cover safe ZIP paths and symlink handling.
- Done: tests cover file hash mismatch and partial result reporting.
- Done: docs explicitly say CTFd-compatible migration is not a goal.
- Done: focused e2e smoke covers ZIP export, async import job polling, result success, and restored file download through filesystem storage.

Residual notes:

- Full S3/SeaweedFS restore smoke is still useful before release-level deployment validation, but it is not required for the backend TODO closure.
- `conflict_mode=skip` is intentionally documented as users/teams-specific; other tables keep their current upsert/do-nothing behavior.

## P2: notifications-tracking-parity

Analyze reference: `PRD/analyze.md` -> `Notifications and tracking`, `Admin statistics funnel`.

Implementation status: done. AstroCTFb now keeps this slice Astro-native rather
than CTFd-compatible: global announcements have a timestamp-cursor count endpoint,
current users have an unread-count endpoint, admins can deliver personal and
team fan-out inbox notifications, and tracking cleanup reports deleted row counts.

Current issue:

- AstroCTFb has WebSocket/SSE and notification routes, which is a backend advantage.
- Persisted notification history already existed; this pass clarified count,
  targeting, team delivery, and cleanup semantics.
- Tracking already feeds admin funnel analytics. Raw `challenge_opens` still records
  repeated page opens, but read-side stats use distinct/min semantics and cleanup
  retention handles storage pressure.

Implementation surfaces:

- Notification usecase/routes.
- WebSocket/SSE event contracts.
- Tracking and challenge-open repositories.
- Admin user/team/challenge detail views and statistics queries.
- Cleanup/retention jobs.

Expected result:

- Notifications and tracking have clear storage, targeting, count, retention, and admin-read contracts.
- Challenge-open tracking is usable by statistics-funnel without leaking private data.
- Challenge-open writes are idempotent per `user_id + challenge_id`, while different users on the same team can still produce separate funnel signals.

Acceptance criteria:

- Done: tests cover global notification creation and user-targeted notification creation.
- Done: tests cover team-targeted fan-out to current non-banned team members, banned-team rejection, missing-team error propagation, and insert failure handling.
- Done: tests cover global notification count via `since_created_at` and user unread count usecase behavior.
- Done: cleanup tests cover challenge-open and tracking retention deleted-row counts, nil repo behavior, first-delete error behavior, and second-delete partial count preservation.
- Done: statistics funnel uses tracking without leaking private data, covered by the already completed `statistics-funnel` target.

Residual notes:

- Admin tracking lookup was already present through user tracking routes; no CTFd-compatible lookup namespace is planned.
- Done: write-side challenge-open dedupe is implemented through the base index migration with a unique `user_id + challenge_id` index.

## P3: custom-field-richness

Analyze reference: `PRD/analyze.md` -> `Users, teams, bans, brackets, custom fields`.

Implementation status: done for v1 metadata/privacy/editability. AstroCTFb now
keeps custom field values as strings, but fields have organizer-facing
`description`, `public`, and `editable` metadata.

Current issue:

- AstroCTFb already had user/team custom fields for registration, but field
  definitions were too thin for organizer workflows.
- Public profile exposure needed an explicit privacy rule instead of leaking all
  values or exposing none.
- Self-edit needed partial updates so changing one field would not erase other
  values.

Implementation surfaces:

- Field schema, sqlc queries, domain/repo mapping, and backup import/export.
- Admin field create/update OpenAPI contracts and response mapping.
- `/auth/me`, `/users/{ID}`, and `PATCH /auth/me` profile contracts.
- Field validator and partial field-value upsert repository behavior.

Expected result:

- Admin-created fields can include a description, public visibility flag, and
  participant editability flag.
- Current user responses include the user's own custom field values.
- Public user profiles include only values whose field has `public=true`.
- `PATCH /auth/me` updates only provided editable user fields and does not delete
  omitted values.

Acceptance criteria:

- Done: field repo integration tests cover metadata create/get/update.
- Done: field-value repo integration tests cover partial upsert without deleting omitted values.
- Done: backup field import covers `description`, `public`, and `editable`.
- Done: settings tests cover field metadata mapping, invalid field/entity types, select options, and non-select option clearing.
- Done: validator tests cover editable-only partial updates, unknown fields, and required text emptiness.
- Done: user usecase tests cover public/private profile filtering and custom field profile patch.
- Done: request/response tests cover custom field metadata and profile custom field mapping.

Residual notes:

- JSON/object custom field values are covered by `custom-field-json-values`.
- Team field edit/profile is now covered by `team-custom-field-self-edit-profile`.

## P3: team-custom-field-self-edit-profile

Analyze reference: `PRD/analyze.md` -> `Users, teams, bans, brackets, custom fields`.

Implementation status: done for v1 team custom field profile flow. AstroCTFb now
applies the same privacy/editability metadata to typed team public/self profile
behavior.

Current issue:

- Team custom field definitions existed, but there was no player/team profile path
  to expose public values or let captains update editable team values.
- The team name update path was tied to team-switch rules; custom-field-only edits
  needed a narrower captain-owned profile update path.

Implementation surfaces:

- Team read/self usecase contracts now return profile wrappers with custom fields.
- `/teams/{ID}` returns public team custom fields only.
- `/teams/my` returns public or editable team custom fields for the caller's own team.
- `PATCH /teams/me` accepts optional `name` and optional `custom_fields`.
- Field-only team patches require captainship and ban checks, but skip team-switch guard.
- Name patches still keep the existing team-switch guard and scoreboard invalidation semantics.
- OpenAPI team request/response schemas include `custom_fields`; generated code and Wire are updated.

Expected result:

- Public team profile values cannot leak non-public custom fields.
- Team members can see fields that are either public or captain-editable in their own team view.
- Captains can patch only editable team custom fields without deleting omitted values.
- Empty team patch requests are rejected as validation errors.

Acceptance criteria:

- Done: usecase tests cover public-only team profile filtering.
- Done: usecase tests cover own-team public-or-editable field visibility.
- Done: usecase tests cover custom-field-only patch without invoking team-switch guard.
- Done: usecase tests cover same-name plus custom-field patch without invoking team-switch guard.
- Done: usecase tests cover validator rejection for non-editable/invalid team fields.
- Done: request tests cover name-only, custom-fields-only, and empty patch mapping.
- Done: OpenAPI validation and Wire generation pass.

Residual notes:

- `GET /teams` intentionally does not include custom fields to avoid N+1 reads and payload growth.
- `GET /teams/{ID}` checks hidden/banned visibility before loading public custom fields.
- Typed team custom field values are covered by `custom-field-json-values`.

## P3: custom-field-json-values

Analyze reference: `PRD/analyze.md` -> `Users, teams, bans, brackets, custom fields`.

Implementation status: done for Astro-native typed values. Field values remain
stored in `field_values.value` as canonical strings, while public API
`custom_fields` values are typed by field definition.

Current issue:

- String-only `custom_fields` made object/array organizer metadata awkward and
  forced number/boolean fields through string transport semantics.
- CTFd is only a capability benchmark here; this is an Astro-native typed API,
  not a CTFd-compatible field-entry contract.

Implementation surfaces:

- Field enum and base migration allow `field_type=json`.
- Register/profile/team `custom_fields` OpenAPI schemas now allow typed JSON values.
- `FieldValidator` normalizes typed API values into canonical string storage.
- User/team read flows decode stored values by field type before response mapping.
- Backup field enum accepts `json`; field-value archive data remains canonical storage strings.

Expected result:

- `text` and `select` values are strings.
- `number` values are integral JSON numbers and are returned as numbers.
- `boolean` values are JSON booleans and are returned as booleans.
- `json` values accept any JSON value; required JSON fields reject `null`.

Acceptance criteria:

- Done: settings validator tests cover typed input, normalized storage output, JSON object values, and required JSON null rejection.
- Done: user/team usecase tests cover typed read-side decoding and canonical write-side storage.
- Done: request/response mapper tests cover typed `custom_fields`.
- Done: integration tests cover `json` field creation and backup field import.
- Done: OpenAPI validation passes with typed `custom_fields`.

## P3: challenge-metadata-envelope-v1

Analyze reference: `PRD/analyze.md` -> `Challenge and flag model`, `CTFd namespace parity`.

Implementation status: done for v1 challenge metadata. AstroCTFb keeps its
current single-answer challenge model, but the organizer-facing challenge
contract now carries `attribution` and an optional `next_id` pointer.

Current issue:

- Challenge authoring already had `connection_info` and `position`, but
  exported/imported/API challenge metadata did not include author/source
  attribution or a next-challenge navigation pointer.
- This is useful as product metadata, not as a CTFd-compatible challenge logic
  clone.

Implementation surfaces:

- Base schema/index migrations and sqlc challenge queries.
- Challenge domain/usecase/repository create, update, list, detail, and backup
  read paths.
- Admin create/update OpenAPI request contracts and public/detail response
  contracts.
- Astro-native backup JSON and CSV import/export paths.

Expected result:

- Admin create/update can set or clear `next_id`.
- Create/update validate that a non-null `next_id` points to an existing
  challenge.
- Update rejects self-references.
- Player responses include metadata only when the challenge itself is visible;
  locked/anonymized responses do not leak attribution or next navigation.
- Backup JSON/CSV round-trip preserves `attribution` and `next_id`.

Acceptance criteria:

- Done: request mapping tests cover create metadata, update value, and update
  null clear semantics.
- Done: response tests cover visible metadata and locked/anonymized redaction.
- Done: usecase tests cover create metadata, self-reference rejection,
  missing-target rejection, and clearing `next_id`.
- Done: integration tests cover repository metadata round-trip and backup
  challenge import with `attribution`/`next_id`.
- Done: sqlc and OpenAPI verification pass.

Residual notes:

- `next_id` is a navigation pointer only; it does not implement challenge logic,
  unlock conditions, graph traversal, or cycle detection.
- Advanced flag groups, partial submissions, and `any/all/team` logic remain in
  `PRD/challenge-answer-model-v2.md`.

## P3: challenge-topics-v1

Analyze reference: `PRD/analyze.md` -> `Challenge and flag model`, `CTFd namespace parity`, `Prioritized decisions`.

Implementation status: done for Astro-native v1. AstroCTFb now has an
admin-only normalized topic taxonomy for organizer classification. This is a
product taxonomy surface, not a CTFd-compatible API clone and not a public
player filter yet.

Current issue:

- Tags already existed, but they are a lightweight challenge labeling surface.
- Organizers may need a normalized topic dictionary for consistent internal
  classification and backup portability.
- The frontend is unfinished, so this pass keeps player-facing topic discovery
  out of scope.

Implementation surfaces:

- Base schema/index migrations add `topics` and `challenge_topics`.
- sqlc and persistent repository cover topic CRUD plus challenge assignment
  reads/writes.
- Challenge usecase validates challenge existence, deduplicates topic IDs, and
  rejects unknown topics inside the transaction before replacing assignments.
- Admin REST/OpenAPI routes expose topic CRUD and challenge-topic assignment.
- Backup JSON import/export preserves `topics` and per-challenge `topic_ids`.

Expected result:

- Admins can create, list, rename, and delete topics.
- Admins can replace a challenge's topic set with a deduplicated list.
- Deleting a challenge or topic cascades assignment cleanup.
- Player challenge list/detail responses do not expose topic data yet.
- Astro-native backups round-trip topic dictionaries and challenge bindings.

Acceptance criteria:

- Done: usecase tests cover create/update, missing name, challenge-not-found,
  replace/dedupe, clear, and invalid topic validation.
- Done: request/response tests cover topic payload mapping and invalid UUID
  conversion.
- Done: integration tests cover topic repository CRUD, assignment replacement,
  cascade cleanup, and backup import/export topic round-trip.
- Done: OpenAPI, sqlc, Wire, mocks, lint, and whitespace checks pass.

Residual notes:

- There is still no public topic filter/search API. Add it only after the
  frontend challenge browsing model is settled.
- Tags and topics intentionally stay separate: tags remain lightweight labels;
  topics are normalized organizer taxonomy.
- CSV backup/import was not extended in this pass; JSON ZIP backup is the
  authoritative Astro-native archive contract.
