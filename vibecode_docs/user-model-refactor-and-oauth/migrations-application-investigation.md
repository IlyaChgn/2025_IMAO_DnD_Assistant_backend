# Migrations Application Investigation

> All facts verified from the repository source code on the `map-editor` branch (HEAD `e167413`).
> No code was modified.

---

# Overview

The project uses [`golang-migrate/migrate/v4`](https://github.com/golang-migrate/migrate) for PostgreSQL schema migrations. Migration SQL files are embedded into the binary at compile time via `//go:embed` and applied via a CLI flag. The migration engine tracks the current version in a `schema_migrations` table with a single-row `(version, dirty)` layout.

There are currently **10 migration versions** (001–010), each with an `.up.sql` and `.down.sql` file in `db/migrations/`.

---

# How migrations are invoked (flags/commands)

## Entry point

`cmd/app/main.go:31-33` — the `-migrate` flag is a string:

```
var migrationsFlag = flag.String("migrate", "", "Run in migrations mode")
```

When non-empty, it calls `migrator.ApplyMigrations(*migrationsFlag)` **before** the server starts.

## Accepted values

The flag value is interpreted in `db/migrator.go:44-52`:

| Value | Behavior | Library call |
|-------|----------|-------------|
| `"latest"` | Apply all pending UP migrations to the highest version | `migrator.Up()` |
| `"<number>"` (e.g. `"10"`) | Migrate to exactly that version (up or down) | `migrator.Migrate(uint(version))` |
| Non-numeric string | Fails with `"invalid migration version"` | — |

## Example commands

```bash
# Dev mode: apply all pending migrations, then start server
go run cmd/app/main.go -migrate latest

# Dev mode: migrate to version 8 specifically, then start server
go run cmd/app/main.go -migrate 8

# Prod mode: apply all pending migrations, then start server
go run cmd/app/main.go -prod -migrate latest
```

## Execution flow after migration

After `ApplyMigrations` returns, **the server starts normally** (`srv.Run()`). If migration fails, `ApplyMigrations` calls `log.Fatal()`, which terminates the process — the server never starts.

Source: `cmd/app/main.go:31-43`, `db/migrator_cmd.go:17-48`

## Important: no migration-only mode

There is no `os.Exit()` or `return` after a successful migration. The server **always** starts after migration completes. To run migrations without starting the server, the process would need to be killed after migration (or the code modified to add such a mode).

---

# Migration engine & version tracking

## Library

`github.com/golang-migrate/migrate/v4` — vendored in the repository.

## Source driver

`iofs` — reads from Go's `embed.FS`:

```go
//go:embed migrations/*.sql
var migrationsFS embed.FS
```

Source: `db/migrator_cmd.go:14-15`

The embed pattern `migrations/*.sql` captures all `.sql` files in `db/migrations/`. At build time, these files are compiled into the binary.

## File name parsing

The library's regex (`source/parse.go:22`):

```
^([0-9]+)_(.*)\.(up|down)\.(.*)$
```

Example: `008_user_profile_status.up.sql` → version `8`, identifier `user_profile_status`, direction `up`, extension `sql`.

The **version number** is the leading digits. The identifier (name) is informational only — it does not affect ordering.

## Migration ordering

`source/migration.go:68-76` — all discovered versions are sorted numerically in ascending order via `sort.Slice`. The `Next(version)` and `Prev(version)` methods use binary search on this sorted index. Ordering is **strictly by numeric version**, not by filename string sort.

## Database driver

`postgres` driver with default configuration. No custom `MigrationsTable` name is set in `db/migrator.go:32`:

```go
driver, err := postgres.WithInstance(db, &postgres.Config{})
```

This means the default table name `schema_migrations` is used (`postgres/postgres.go:35`).

## Version tracking table

Table: `public.schema_migrations`

```sql
CREATE TABLE IF NOT EXISTS "schema_migrations" (
    version bigint NOT NULL PRIMARY KEY,
    dirty boolean NOT NULL
);
```

The table always has **zero or one rows**. On each migration step:
1. The row is truncated
2. A new row `(target_version, true)` is inserted (dirty = true)
3. The SQL file is executed
4. The row is updated to `(target_version, false)` (dirty = false)

This truncate-insert-update happens inside a transaction (`postgres.go:357-390`).

## Concurrency protection

The postgres driver uses **PostgreSQL advisory locks** (`pg_advisory_lock`) to prevent concurrent migration runs (`postgres.go:234-250`). The lock is held for the entire migration session and released when done.

## How to check the current version

```sql
SELECT version, dirty FROM schema_migrations;
```

If the table is empty, no migrations have been applied (version = nil / -1 internally).

---

# Scenario: DB at 007, repo has 008-010, run `migrate latest`

## What happens step-by-step

Command: `go run cmd/app/main.go -migrate latest`

1. `ApplyMigrations("latest")` is called (`migrator_cmd.go:38`).
2. `applyMigrations(db, "latest")` calls `migrator.Up()` (`migrator.go:45`).
3. `Up()` acquires advisory lock (`migrate.go:268`).
4. `Up()` reads current version from `schema_migrations` → `(7, false)` (`migrate.go:272`).
5. `Up()` checks dirty flag — not dirty, so proceed (`migrate.go:277-279`).
6. `readUp(7, -1, ret)` is called with limit=-1 (no limit) (`migrate.go:283`).
7. `readUp` calls `sourceDrv.Next(7)` → returns `8`.
8. Migration `008_user_profile_status.up.sql` is queued.
9. `sourceDrv.Next(8)` → returns `9`.
10. Migration `009_user_identity.up.sql` is queued.
11. `sourceDrv.Next(9)` → returns `10`.
12. Migration `010_drop_vkid.up.sql` is queued.
13. `sourceDrv.Next(10)` → returns `ErrNotExist` (no more migrations). Channel is closed.
14. `runMigrations` processes each migration **sequentially** from the channel:

**For each migration N (8, 9, 10):**
- a. `SetVersion(N, true)` — writes `(N, true)` to `schema_migrations` (within a transaction).
- b. `Run(body)` — executes the SQL file content against PostgreSQL.
- c. `SetVersion(N, false)` — writes `(N, false)` to `schema_migrations` (within a transaction).

15. After all 3 migrations succeed, advisory lock is released.
16. `ApplyMigrations` logs "All migrations have been applied" and returns.
17. Server starts.

## Key guarantees

- **Strictly sequential**: 008 → 009 → 010, one at a time. Never parallel.
- **NOT in a single transaction**: Each migration file is executed as a single statement (or multi-statement if enabled). The `SetVersion` calls are in their own transactions. But the SQL body of each migration file runs **without an explicit transaction wrapper** — it's a raw `ExecContext` call (`postgres.go:298`).
- **Each migration is independently committed to `schema_migrations`**: If 008 succeeds and 009 fails, the DB version will be `9` with `dirty=true` (see Failure section below).

## What this means for our migrations specifically

| Step | Migration | Key DDL | Risk |
|------|-----------|---------|------|
| 1 | 008 | Renames `name`→`display_name`, `avatar`→`avatar_url`, adds `status`, `role`, `email`, `email_verified` | **Breaking**: old binary queries fail after this |
| 2 | 009 | Creates `user_identity` table, backfills from `vkid`, makes `vkid` nullable | Safe if 008 succeeded |
| 3 | 010 | Drops `vkid` column | Safe if 009 succeeded; **irreversible** data-wise if VK identities weren't backfilled |

---

# Failure/dirty behavior

## What happens if migration 009 fails after 008 succeeded

1. Migration 008 completes: `schema_migrations = (8, false)`.
2. Migration 009 starts: `schema_migrations = (9, true)` (dirty flag set BEFORE SQL runs).
3. SQL execution fails (e.g., syntax error, constraint violation).
4. `runMigrations` returns the error immediately. No more migrations are processed.
5. `schema_migrations` remains at `(9, true)` — **dirty state**.
6. `ApplyMigrations` calls `log.Fatal` → process exits. Server does NOT start.

## The dirty state problem

If the database is in dirty state `(9, true)`, **all subsequent migration commands will refuse to run**:

```
Dirty database version 9. Fix and force version.
```

This is because `Up()`, `Migrate()`, `Steps()`, and `Down()` all check dirty at the start (`migrate.go:224-226, 277-279`) and return `ErrDirty` immediately.

## Recovery from dirty state

The `golang-migrate` library provides a `Force(version int)` method (`migrate.go:367-381`) that sets the version to a specified value with `dirty=false`, without running any SQL. However, **this project does not expose `Force` through its CLI flags**.

### Manual recovery options

**Option 1: Direct SQL**

```sql
-- Check current state
SELECT version, dirty FROM schema_migrations;

-- If dirty=true at version 9 and the 009 SQL partially applied:
-- 1. Manually fix/revert the partial changes
-- 2. Force the version back to the last clean state:
UPDATE schema_migrations SET version = 8, dirty = false;

-- Or if the migration actually completed but SetVersion failed:
UPDATE schema_migrations SET dirty = false;
```

**Option 2: Drop and recreate**

```sql
-- Nuclear option: reset migration tracking
DROP TABLE schema_migrations;
-- Then re-run: go run cmd/app/main.go -migrate <correct_version>
```

### Important: DDL statements are not transactional in the usual sense

PostgreSQL DDL (`ALTER TABLE`, `CREATE TABLE`, `DROP COLUMN`) **is** transactional, but the migration engine does NOT wrap each file in a transaction. Each file is executed as a single `ExecContext` call. If a file contains multiple statements, some may succeed before one fails, leaving a partially-applied migration. For our migrations:

- `008`: 2 `ALTER RENAME` + 1 `ALTER ADD COLUMN` — if the last fails, renames are already committed.
- `009`: `CREATE TABLE` + `INSERT` backfill + `ALTER COLUMN` — if backfill fails, table exists but is empty.
- `010`: Single `DROP COLUMN IF EXISTS` — atomic.

---

# Target version & rollback

## Migrating to a specific version

```bash
go run cmd/app/main.go -migrate 8
```

This calls `migrator.Migrate(8)` (`migrate.go:214-232`), which:

1. Reads current version from DB.
2. If current < 8: applies UP migrations from current+1 to 8.
3. If current > 8: applies DOWN migrations from current to 9 (rolling back until version=8).
4. If current == 8: returns `ErrNoChange` (handled gracefully — not fatal).

## Going down (rollback)

The `Migrate(version)` function calls `read(from, to, ret)` (`migrate.go:229`). When `from > to`, `read` calls `sourceDrv.Prev()` repeatedly and queues DOWN migrations (`migrate.go:481-526`).

Example: DB is at version 10, command is `-migrate 7`:

1. `read(10, 7)` → going down
2. Applies `010_drop_vkid.down.sql` (version 10 → 9)
3. Applies `009_user_identity.down.sql` (version 9 → 8)
4. Applies `008_user_profile_status.down.sql` (version 8 → 7)

Each down migration reverses the up migration.

## Rollback hazards for our specific migrations

| Rollback | Down file | Risk |
|----------|-----------|------|
| 010 → 009 | Restores `vkid` column, backfills from `user_identity` | **Fails** if VK identities were deleted via `unlink` — those users get NULL vkid, then `SET NOT NULL` fails |
| 009 → 008 | Restores `vkid NOT NULL`, drops `user_identity` table | **Fails** for Google/Yandex-only users (no vkid to restore) |
| 008 → 007 | Drops new columns, renames back | Safe if no code depends on new column names |

## What if target version doesn't exist?

If you pass `-migrate 15` and there's no migration 15:

- `Migrate(15)` calls `read(current, 15, ret)`.
- `read` checks `versionExists(15)` — looks for any `.up.sql` or `.down.sql` with version 15.
- Not found → returns error → `log.Fatal`.

---

# Recommended deploy procedure (stop → migrate → start)

Based on the repository's actual code and migration engine behavior:

## Pre-deploy checklist

1. **Check current DB version:**
   ```sql
   SELECT version, dirty FROM schema_migrations;
   ```
   Expected: `(7, false)`. If dirty=true, fix before proceeding (see Recovery section).

2. **Review pending migrations:**
   - `db/migrations/008_user_profile_status.up.sql` — column renames + new columns
   - `db/migrations/009_user_identity.up.sql` — new table + backfill
   - `db/migrations/010_drop_vkid.up.sql` — drop column

3. **Backup the database** (standard practice, especially before 008 which renames columns).

## Deploy steps

```bash
# 1. Stop the running server
#    (kill the process, stop the systemd unit, etc.)

# 2. Apply migrations (dev mode example)
go run cmd/app/main.go -migrate latest
# This will:
#   - Apply 008, 009, 010 sequentially
#   - Log "All migrations have been applied"
#   - Then START the server automatically

# OR, if you want to migrate without starting the server,
# you must kill the process after migration logs success.
# There is no migrate-only flag in the current code.
```

**Alternative: step-by-step approach** (safer for the first time):

```bash
# Apply one at a time and verify between each
go run cmd/app/main.go -migrate 8
# Verify: SELECT version, dirty FROM schema_migrations; → (8, false)
# Kill the server (it will start after migration)

go run cmd/app/main.go -migrate 9
# Verify: SELECT version, dirty FROM schema_migrations; → (9, false)
# Kill the server

go run cmd/app/main.go -migrate 10
# Verify: SELECT version, dirty FROM schema_migrations; → (10, false)
# Server is now running with all migrations applied
```

## Post-deploy verification

```sql
-- 1. Check version
SELECT version, dirty FROM schema_migrations;
-- Expected: (10, false)

-- 2. Verify 008: column renames
SELECT column_name FROM information_schema.columns
WHERE table_name = 'user' AND table_schema = 'public'
ORDER BY ordinal_position;
-- Should include: display_name, avatar_url, status, role, email, email_verified
-- Should NOT include: name, avatar

-- 3. Verify 009: identity table exists
SELECT COUNT(*) FROM user_identity;
-- Should return the number of backfilled VK identities

-- 4. Verify 010: vkid column is gone
SELECT column_name FROM information_schema.columns
WHERE table_name = 'user' AND column_name = 'vkid';
-- Should return 0 rows
```

---

# Checklist

## Migration files consistency

| Version | Up file | Down file | Pair complete | Names consistent |
|:-------:|---------|-----------|:---:|:---:|
| 001 | `001_test.up.sql` | `001_test.down.sql` | Yes | Yes |
| 002 | `002_user.up.sql` | `002_user.down.sql` | Yes | Yes |
| 003 | `003_encouter.up.sql` | `003_encounter.down.sql` | Yes | **No** — typo in up file (`encouter` vs `encounter`). Does not affect functionality (identifier is informational only). |
| 004 | `004_encounter_uuid.up.sql` | `004_encounter_uuid.down.sql` | Yes | Yes |
| 005 | `005_removed_encounter_id.up.sql` | `005_removed_encounter_id.down.sql` | Yes | Yes |
| 006 | `006_maps.up.sql` | `006_maps.down.sql` | Yes | Yes |
| 007 | `007_user_timestamps.up.sql` | `007_user_timestamps.down.sql` | Yes | Yes |
| 008 | `008_user_profile_status.up.sql` | `008_user_profile_status.down.sql` | Yes | Yes |
| 009 | `009_user_identity.up.sql` | `009_user_identity.down.sql` | Yes | Yes |
| 010 | `010_drop_vkid.up.sql` | `010_drop_vkid.down.sql` | Yes | Yes |

## Embed pattern coverage

The pattern `migrations/*.sql` in `db/migrator_cmd.go:14` matches all files in `db/migrations/` with `.sql` extension. All 20 files (10 up + 10 down) are captured.

## File name parsing verification

All files match the regex `^([0-9]+)_(.*)\.(up|down)\.(.*)$`:
- Version numbers are 3-digit zero-padded (`001`–`010`), parsed as `uint` → `1`–`10`.
- Direction is correctly `up` or `down` in all filenames.
- Extension is `sql` for all files.

## Gap check

No version gaps in the sequence 1–10. The `Next()` function uses sorted index lookup, so gaps would be handled correctly by the library regardless, but there are none.

## Known risks

| Risk | Severity | Description |
|------|----------|-------------|
| No `Force` CLI command | Medium | If dirty state occurs, recovery requires direct SQL on `schema_migrations`. Consider adding `-migrate force <version>` to `migrator.go`. |
| No migrate-only mode | Low | After migration, server always starts. For deploy scripts that need migrate→verify→start, the server must be killed after migration or the code needs a migrate-only flag. |
| DDL not wrapped in transaction | Medium | Partial migration failures can leave schema in inconsistent state. The postgres driver does not wrap each file in `BEGIN/COMMIT`. However, PostgreSQL's transactional DDL means single-statement files are safe. Multi-statement files (008, 009) carry partial-apply risk. |
| 003 filename typo | None | `003_encouter.up.sql` vs `003_encounter.down.sql` — does not affect version ordering or migration behavior. Cosmetic only. |
