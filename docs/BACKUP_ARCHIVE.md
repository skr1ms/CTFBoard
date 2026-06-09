# Backup Archive Contract

AstroCTFb backup archives are Astro-native operational artifacts. They are not
CTFd-compatible migration files, and the importer should not accept CTFd archive
shapes unless a separate product decision introduces a dedicated converter.

## Format

The archive is a ZIP file with a required `backup.json` entry at the archive
root. `backup.json` is the source of truth for database records and must be at
most 100 MiB when imported. Exported ZIP archives also include `README.md` for
operator context, but the importer does not require it.

Current backup version: `1.0`.

The importer requires an exact version match with the backend's
`domain.BackupVersion`. Any incompatible archive shape change must bump that
constant, update OpenAPI/docs, and add import/export regression tests.

File payloads are optional and live under deterministic paths derived from the
file type:

```text
files/challenge-<challenge_id>/<filename>
files/writeup-<challenge_id>/<filename>
files/page-<page_id>/<filename>
```

The `files[].location` value in `backup.json` remains the internal object
storage path. Import validates that location before writing, and then restores
the payload to that exact path.

## Data Model

`backup.json` uses the `BackupData` schema exposed by OpenAPI. It contains:

- `version`, `exported_at`, and the singleton `competition` record.
- Challenge content, challenge tags, hints, requirements, and official
  solutions.
- Static pages authored by admins.
- Optional teams, users, awards, solves, hint unlocks, file metadata, comments,
  custom fields, field values, ratings, brackets, and tags.

JSON export and ZIP export share the same `backup.json` shape. ZIP export adds
file payloads when `include_files=true`.

## Import Semantics

`POST /admin/import` starts an asynchronous ZIP import job and returns
`202 Accepted` with the job status record. Operators poll
`GET /admin/import/jobs/{ID}` until the status becomes `completed` or `failed`.

Job statuses are `queued`, `running`, `completed`, and `failed`. Job phases are
`queued`, `validating`, `importing_db`, `restoring_files`, `cleanup`, and
`finished`.

Import options:

- `erase_existing`: truncates backup-managed tables before replaying records.
- `conflict_mode`: currently `overwrite` only for ZIP imports.
- `validate_files`: validates SHA-256 for restored payloads before accepting
  file metadata.
- `preserve_admin_roles`: keeps imported admin roles; otherwise imported admin
  accounts are downgraded to participant users.

Database records are imported in one transaction. The restore order is:
competition, tags, topics, challenges, challenge tags, challenge topics,
brackets, pages, users, teams, user team membership links, awards, solves,
hint unlocks, file metadata, challenge
requirements, solutions, ratings, comments, fields, and field values.

File payload upload happens after the database transaction commits because
object storage is not transactional. Missing payloads, invalid storage
locations, symlink payloads, and SHA-256 mismatches are skipped before database
import, so rejected file metadata is not imported. Upload failures after commit
are returned as warnings with `skipped_count` increments.

Startup recovery marks interrupted queued or running import jobs as failed so
operators do not see stale in-progress imports after a restart.
