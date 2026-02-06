# User Entity Refactor — Implementation Log

## Status

| PR | Title | Commit | Status |
|----|-------|--------|--------|
| PR1+2 | Cookie flags + User timestamps | `aaa7c42` | DONE |
| PR3a | Schema rename + status column (no API break) | `98720f1` | DONE |
| PR3b | JSON breaking rename | `bc6b44e` | DONE |
| PR4 | user_identity table + VK backfill | `0189cab` | DONE |
| PR5 | Drop vkid | `a1b4ac4` | DONE |
| PR6 | OAuthProvider abstraction | `79cf0dc` | DONE |
| PR7 | Google OAuth | `0d26c23` | DONE |
| PR8 | Yandex OAuth | `c6880d4` | DONE |
| PR9 | Manual linking endpoints | `580a522` | DONE |

---

# PR1+2 — Cookie flags + User timestamps

- **Commit:** `aaa7c42`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** `007_user_timestamps` (up/down)
- **Public API changes:** no
- **Breaking change:** no
- **Requires frontend changes:** no
- **Requires prod downtime:** no

## Summary
Added `HttpOnly`, `Secure`, `SameSite=Lax` cookie flags. Created migration 007 adding `created_at`, `updated_at`, `last_login_at` timestamp columns to `public.user`. Added `UpdateLastLoginAt` to `AuthRepository`. Best-effort `UpdateLastLoginAt` call after successful login. Session duration moved to config (`config.yaml` + `config.go`).

## Files changed
- `db/migrations/007_user_timestamps.up.sql` — ADD COLUMN created_at, updated_at, last_login_at
- `db/migrations/007_user_timestamps.down.sql` — DROP those columns
- `internal/pkg/auth/delivery/auth_handlers.go` — cookie flags
- `internal/pkg/auth/delivery/session.go` — NEW: `createSession`, `clearSession` helpers
- `internal/pkg/auth/delivery/auth_handlers_test.go` — cookie flag tests
- `internal/pkg/auth/interfaces.go` — added `UpdateLastLoginAt`
- `internal/pkg/auth/repository/auth_queries.go` — `UpdateLastLoginAtQuery`
- `internal/pkg/auth/repository/auth_storage.go` — `UpdateLastLoginAt` implementation
- `internal/pkg/auth/usecases/auth.go` — call `UpdateLastLoginAt` after login
- `internal/pkg/auth/usecases/auth_test.go` — test for `UpdateLastLoginAt`
- `internal/pkg/config/config.go` — session duration config
- `internal/pkg/config/config.yaml` — session section
- `internal/pkg/server/app.go` — pass session duration
- `internal/pkg/server/delivery/routers/router.go` — pass `isProd`

## Behavior / API changes
- Cookies now have `HttpOnly`, `SameSite=Lax`; `Secure` flag set only in production mode
- `last_login_at` updated on each login (best-effort, failure does not block login)
- No response body changes

## Tests
- `TestLoginCookieFlags` (dev + prod mode)
- `TestLogoutCookieFlags` (dev + prod mode)
- `TestLogin` updated to expect `UpdateLastLoginAt` mock call

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- PR1 (cookie flags) and PR2 (timestamps) combined into one commit because they are small and independent
- `last_login_at` is nullable (NULL for users who never logged in after migration)
- Cookie `MaxAge` set via config `session.duration`

## Rollout notes / Risks
- Migration 007 is additive (ADD COLUMN with defaults) — safe to run online
- Cookie flag change is transparent to frontend
- No breaking changes

## Backlog updates
None

---

# PR3a — Schema rename + status column

- **Commit:** `98720f1`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** `008_user_profile_status` (up/down)
- **Public API changes:** yes (new `status` field in User JSON; new `USER_INACTIVE` 403 error)
- **Breaking change:** no (additive field; new error code on existing endpoints)
- **Requires frontend changes:** yes (handle `USER_INACTIVE` 403)
- **Requires prod downtime:** yes — **hard boundary**: migration 008 renames DB columns `name -> display_name`, `avatar -> avatar_url`. Old binary queries reference old column names and will fail after migration.

## Summary
Migration 008 renames DB columns (`name` -> `display_name`, `avatar` -> `avatar_url`) and adds `status`, `role`, `email`, `email_verified` columns. Go repository SQL queries updated to match renamed columns. Added `Status` field to `models.User` (JSON tag: `status,omitempty`). Added `LoginRequiredMiddleware` check: returns 403 `USER_INACTIVE` for users with `status != "active"`.

Note: Go struct field names remained `Name`/`Avatar` at this stage — the Go rename + JSON tag change happened in PR3b.

## Files changed
- `db/migrations/008_user_profile_status.up.sql` — RENAME COLUMN name->display_name, avatar->avatar_url; ADD status, role, email, email_verified
- `db/migrations/008_user_profile_status.down.sql` — DROP new columns, RENAME back
- `internal/models/auth.go` — added `Status` field
- `internal/pkg/auth/repository/auth_queries.go` — SQL now uses `display_name`, `avatar_url`
- `internal/pkg/auth/repository/auth_storage.go` — Scan updated for new column names
- `internal/pkg/middleware/auth/login_required.go` — check `user.Status`, return 403 `USER_INACTIVE`
- `internal/pkg/middleware/auth/login_required_test.go` — NEW: tests for inactive user
- `internal/pkg/server/delivery/responses/responses.go` — added `ErrUserInactive`

## Behavior / API changes
- New JSON field `status` (omitempty) in all responses embedding `User` — additive, non-breaking
- New 403 response `{"status": "USER_INACTIVE"}` on all protected routes when `user.status` is not `"active"`
- DB column names changed — **binary and migration must deploy together**

## Tests
- `TestLoginRequiredMiddleware` covers: no cookie, invalid session, inactive user, active user

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- RENAME COLUMN chosen over alias/view — simpler, but creates hard deploy boundary
- `status` default is `'active'` — existing users unaffected
- `role`, `email`, `email_verified` added for future use, not exposed in API yet

## Rollout notes / Risks
- **Hard boundary:** migration 008 and PR3a binary must deploy atomically. Old binary will crash with SQL errors after migration.
- Rollback: `go run cmd/app/main.go -migrate 7` + deploy pre-PR3a binary
- 403 `USER_INACTIVE` is a new error code that frontend must handle

## Backlog updates
- `my_docs/Frontend API field changes backlog.md`: USER_INACTIVE entry added

---

# PR3b — Breaking JSON rename

- **Commit:** `bc6b44e`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** none
- **Public API changes:** yes
- **Breaking change:** yes — JSON field renames (`name` -> `displayName`, `avatar` -> `avatarUrl`)
- **Requires frontend changes:** yes — all `user.name` / `user.avatar` reads must update
- **Requires prod downtime:** no (coordinate frontend deploy)

## Summary
Renamed Go struct fields `Name` -> `DisplayName`, `Avatar` -> `AvatarURL` in `models.User`. Changed JSON tags from `name` -> `displayName`, `avatar` -> `avatarUrl,omitempty`. This is a breaking API change for frontend consumers.

## Files changed
- `internal/models/auth.go` — field + tag rename
- `internal/pkg/auth/usecases/auth.go` — field references
- `internal/pkg/auth/delivery/auth_handlers.go` — log line
- `internal/pkg/auth/repository/auth_storage.go` — Scan calls, query params
- `internal/pkg/table/repository/table_manager.go` — field references
- `internal/pkg/auth/delivery/auth_handlers_test.go` — test struct literals
- `internal/pkg/auth/usecases/auth_test.go` — test struct literals
- `internal/pkg/middleware/auth/login_required_test.go` — test struct literals
- `internal/pkg/table/delivery/servews_integration_test.go` — test struct literals + gofmt fix
- `internal/pkg/table/delivery/table_handlers_test.go` — test struct literals
- `internal/pkg/table/usecases/table_test.go` — test struct literals
- `internal/pkg/bestiary/delivery/bestiary_handlers_test.go` — test struct literals
- `internal/pkg/character/delivery/character_handlers_test.go` — test struct literals
- `internal/pkg/encounter/delivery/encounter_handlers_test.go` — test struct literals
- `internal/pkg/maps/delivery/maps_handlers_test.go` — test struct literals
- `internal/pkg/maptiles/delivery/maptiles_handlers_test.go` — test struct literals
- `internal/pkg/auth/mocks/mock_auth.go` — regenerated
- `my_docs/Frontend API field changes backlog.md` — PR3b entries marked Done

## Behavior / API changes
- JSON response field `name` -> `displayName`
- JSON response field `avatar` -> `avatarUrl`
- Frontend MUST update field access accordingly
- Existing Redis sessions with old `"name"` key won't populate `DisplayName` — accepted as pre-production cost

## Tests
All existing tests updated to use new field names. All pass.

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean (after fixing servews_integration_test.go)
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- Combined Go field rename + JSON tag change in one PR (PR3a only added Status field and SQL changes, didn't rename Go fields)
- Accepted Redis session deserialization mismatch as pre-production cost

## Rollout notes / Risks
- **Breaking change**: frontend must deploy updated field names simultaneously
- Old sessions will have empty DisplayName until they expire (30-day TTL)

## Backlog updates
- `my_docs/Frontend API field changes backlog.md`: PR3b rename entries marked Done, USER_INACTIVE marked Done

---

# PR4 — user_identity table + VK backfill

- **Commit:** `0189cab`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** `009_user_identity` (up/down)
- **Public API changes:** no
- **Breaking change:** no
- **Requires frontend changes:** no
- **Requires prod downtime:** no

## Summary
Created `user_identity` table with migration 009. Added `IdentityRepository` interface and PostgreSQL implementation. Refactored Login flow to use identity-based lookup instead of direct VKID check. Backfill migration copies existing `user.vkid` data into identity rows.

## Files changed
- `db/migrations/009_user_identity.up.sql` — CREATE TABLE, backfill from vkid, make vkid nullable
- `db/migrations/009_user_identity.down.sql` — restore vkid NOT NULL, DROP TABLE
- `internal/models/auth.go` — added `UserIdentity` struct
- `internal/pkg/auth/interfaces.go` — added `IdentityRepository` interface
- `internal/pkg/auth/repository/identity_storage.go` — NEW: IdentityRepository implementation
- `internal/pkg/auth/repository/identity_queries.go` — NEW: SQL queries
- `internal/pkg/apperrors/auth.go` — added `IdentityNotFoundError`
- `internal/pkg/auth/usecases/auth.go` — added `identityRepo` field, rewrote Login flow
- `internal/pkg/auth/usecases/auth_test.go` — rewritten with identity mock expectations
- `internal/pkg/server/app.go` — wired `identityRepository`

## Behavior / API changes
- Login flow now: FindByProvider("vk", vkUserID) -> existing user path OR CreateUser + CreateIdentity
- Best-effort UpdateLastUsed on identity after login
- Best-effort UpdateLastLoginAt on user after login
- No external API changes

## Tests
All usecase tests rewritten with MockIdentityRepository. Test cases cover:
- New user creation + identity creation
- Existing user with no profile changes
- Existing user with profile update
- VK API errors, unmarshal errors, session creation errors
- Identity creation error for new user

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean (after fixing identity_queries.go, identity_storage.go)
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- Used `BIGINT GENERATED BY DEFAULT AS IDENTITY` for PK (consistent with existing tables)
- Two UNIQUE constraints: `(provider, provider_user_id)` and `(user_id, provider)` — one user per provider
- Backfill uses `ON CONFLICT DO NOTHING` — idempotent
- `vkid` made nullable (not dropped yet — that's PR5)

## Rollout notes / Risks
- Migration 009 must run before deploying PR4 binary (new code queries `user_identity`)
- No breaking API changes
- Backfill only copies rows where `vkid IS NOT NULL`

## Backlog updates
None

---

# PR5 — Drop vkid from users

- **Commit:** `a1b4ac4`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** `010_drop_vkid` (up/down)
- **Public API changes:** no (vkid was removed from JSON in PR3b)
- **Breaking change:** no (API unchanged; internal-only)
- **Requires frontend changes:** no
- **Requires prod downtime:** no

## Summary
Removed `vkid` column from `public.user` table via migration 010. Removed `VKID` field from `models.User`. Changed `AuthRepository.CheckUser(vkid)` to `GetUserByID(userID int)`. All SQL queries updated to no longer reference vkid.

## Files changed
- `db/migrations/010_drop_vkid.up.sql` — DROP COLUMN vkid
- `db/migrations/010_drop_vkid.down.sql` — restore vkid from user_identity
- `internal/models/auth.go` — removed VKID field from User
- `internal/pkg/auth/interfaces.go` — `CheckUser` -> `GetUserByID`
- `internal/pkg/auth/repository/auth_queries.go` — all queries without vkid
- `internal/pkg/auth/repository/auth_storage.go` — `GetUserByID`, `CreateUser`, `UpdateUser` without vkid
- `internal/pkg/auth/repository/identity_queries.go` — removed `u.vkid` from FindUserByIdentityQuery
- `internal/pkg/auth/usecases/auth.go` — Login uses `GetUserByID(identity.UserID)`
- `internal/pkg/auth/usecases/auth_test.go` — removed VKID from test literals, `CheckUser` -> `GetUserByID`
- `internal/pkg/auth/mocks/mock_auth.go` — regenerated
- `internal/pkg/encounter/repository/encounter_integration_test.go` — removed vkid from test schema

## Behavior / API changes
- No external API changes (VKID was never in JSON responses after PR3b)
- Internal: user lookup now exclusively via `user_identity` table

## Tests
All usecase tests updated. Integration test schema updated.

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- Down migration restores vkid by joining `user_identity` (provider='vk')
- `UpdateUser` now uses `WHERE id = $1` instead of `WHERE vkid = $1`
- Down migration adds UNIQUE and CHECK constraints back on vkid

## Rollout notes / Risks
- Deploy order: PR5 binary first (queries don't reference vkid), then migrate 010 (DROP COLUMN)
  - Alternatively: migrate then deploy — both work because PR4 already stopped referencing vkid in Login
- Down migration fails if `user_identity` rows for provider='vk' were deleted
- Migration 010 must run after 009

## Backlog updates
None

---

# PR6 — OAuthProvider abstraction

- **Commit:** `79cf0dc`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** none
- **Public API changes:** yes
- **Breaking change:** yes — route changed; Redis session format changed (mass logout)
- **Requires frontend changes:** yes — login endpoint URL changed
- **Requires prod downtime:** yes — **hard boundary**: all existing Redis sessions become unreadable after deploy (forced re-login for all users)

## Summary
Introduced `OAuthProvider` interface with `Name()` and `Authenticate()` methods. Refactored VK API to implement it. Login handler now accepts `{provider}` path variable. Auth usecases use a `map[string]OAuthProvider` instead of a single VK client. Added `Provider` field to `FullSessionData`. Created `OAuthResult` model.

## Files changed
- `internal/models/auth.go` — added `OAuthResult` struct, `Provider` field in `FullSessionData`
- `internal/pkg/auth/interfaces.go` — added `OAuthProvider` interface, removed `VKApi`, changed `Login` signature
- `internal/pkg/auth/external/vk_api.go` — implements `OAuthProvider`, `NewVKApi` returns `OAuthProvider`
- `internal/pkg/auth/usecases/auth.go` — providers map, `Login(provider, ...)`, generic flow
- `internal/pkg/auth/usecases/auth_test.go` — rewritten with `MockOAuthProvider`
- `internal/pkg/auth/delivery/auth_handlers.go` — `mux.Vars(r)["provider"]`, new error handling
- `internal/pkg/auth/delivery/auth_handlers_test.go` — updated fake, `mux.SetURLVars`
- `internal/pkg/server/delivery/routers/auth.go` — `/login/{provider}` route
- `internal/pkg/server/app.go` — created providers map, wired all providers
- `internal/pkg/apperrors/auth.go` — added `UnsupportedProviderError`
- `internal/pkg/server/delivery/responses/responses.go` — added `ErrBadRequest`, `ErrOAuthProvider`
- `internal/pkg/auth/mocks/mock_auth.go` — regenerated (includes `MockOAuthProvider`)

## Behavior / API changes
- **BREAKING**: Route changed from `POST /api/auth/login` -> `POST /api/auth/login/{provider}`
- **BREAKING**: Adding `Provider` to `FullSessionData` changes Redis session format — **all existing sessions will be invalidated (mass logout)**
- New error: `UnsupportedProviderError` returns 400 `Bad request` for unknown provider
- Note: `provider` is stored in Redis `FullSessionData` only; it is NOT returned in `AuthResponse` to the client

## Tests
Usecase tests fully rewritten with MockOAuthProvider. Handler tests use `mux.SetURLVars`.

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- Single `OAuthResult` struct returned by all providers (provider-agnostic)
- VK-specific JSON parsing moved into VK provider's `Authenticate`
- Provider stored in session for future identity-management features
- Provider is a URL path variable, not a request body field

## Rollout notes / Risks
- **Users will be logged out** (Redis session format change)
- Frontend must update login endpoint URL to include provider name
- Frontend and backend must deploy together (old frontend calls `/api/auth/login` which no longer exists)
- New providers can be added without changing usecase code
- Rollback: revert binary; clear Redis or accept that new-format sessions won't parse for old binary

## Backlog updates
- Frontend must update: `POST /api/auth/login` -> `POST /api/auth/login/{provider}`

---

# PR7 — Google OAuth provider

- **Commit:** `0d26c23`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** none
- **Public API changes:** yes (new endpoint)
- **Breaking change:** no (additive)
- **Requires frontend changes:** no (only if Google login is to be used)
- **Requires prod downtime:** no

## Summary
Added Google OAuth provider implementing `OAuthProvider` interface. Uses Google's token and userinfo endpoints.

## Files changed
- `internal/pkg/auth/external/google_oauth.go` — NEW: Google OAuth implementation
- `internal/pkg/config/config.go` — added `GoogleOAuthConfig` struct
- `internal/pkg/server/app.go` — wired Google OAuth client
- `internal/pkg/apperrors/auth.go` — added `OAuthProviderError` (generic)
- `internal/pkg/auth/delivery/auth_handlers.go` — error handling for `OAuthProviderError`
- `internal/pkg/server/delivery/responses/responses.go` — added `ErrOAuthProvider`

## Behavior / API changes
- New endpoint: `POST /api/auth/login/google`
- New config section: `google_oauth` with `client_id`, `client_secret`, `redirect_uri`

## Tests
Existing provider tests cover the dispatch logic. Google provider tested through the OAuthProvider interface.

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- Used `googleapis.com/oauth2/v2/userinfo` for user info (simpler than v3)
- Returns `OAuthProviderError` for HTTP errors from Google

## Rollout notes / Risks
- Requires Google OAuth credentials in config/env
- Frontend needs Google OAuth client-side flow to obtain authorization code
- If credentials are missing, the provider still registers but all Authenticate calls will fail

## Backlog updates
None

---

# PR8 — Yandex OAuth provider

- **Commit:** `c6880d4`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** none
- **Public API changes:** yes (new endpoint)
- **Breaking change:** no (additive)
- **Requires frontend changes:** no (only if Yandex login is to be used)
- **Requires prod downtime:** no

## Summary
Added Yandex OAuth provider implementing `OAuthProvider` interface. Uses Yandex OAuth token and login info endpoints.

## Files changed
- `internal/pkg/auth/external/yandex_oauth.go` — NEW: Yandex OAuth implementation
- `internal/pkg/config/config.go` — added `YandexOAuthConfig` struct
- `internal/pkg/server/app.go` — wired Yandex OAuth client

## Behavior / API changes
- New endpoint: `POST /api/auth/login/yandex`
- New config section: `yandex_oauth` with `client_id`, `client_secret`

## Tests
Existing provider tests cover the dispatch logic. Yandex provider tested through the OAuthProvider interface.

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- Avatar URL constructed from `default_avatar_id` using Yandex's `avatars.yandex.net` CDN
- Falls back to `first_name + last_name` if `display_name` is empty

## Rollout notes / Risks
- Requires Yandex OAuth credentials in config/env
- Frontend needs Yandex OAuth client-side flow to obtain authorization code

## Backlog updates
None

---

# PR9 — Manual linking/unlinking endpoints

- **Commit:** `580a522`
- **Branch:** `battle-map-new-features-and-macrotile-editor-and-ws-refactor`
- **Migrations:** none
- **Public API changes:** yes (three new endpoints)
- **Breaking change:** no (additive)
- **Requires frontend changes:** no (only if identity management UI is to be built)
- **Requires prod downtime:** no

## Summary
Added three new authenticated endpoints for managing user identity links: list, link, and unlink. Users can now connect multiple OAuth providers to a single account and disconnect providers (with a minimum of one).

## Files changed
- `internal/pkg/auth/interfaces.go` — added `ListIdentities`, `LinkIdentity`, `UnlinkIdentity` to `AuthUsecases`; added `DeleteByUserAndProvider` to `IdentityRepository`
- `internal/pkg/auth/usecases/auth.go` — implemented `ListIdentities`, `LinkIdentity`, `UnlinkIdentity`
- `internal/pkg/auth/usecases/auth_test.go` — tests for all three new methods
- `internal/pkg/auth/delivery/auth_handlers.go` — added `ctxUserKey` field, `ListIdentities`, `LinkIdentity`, `UnlinkIdentity` handlers
- `internal/pkg/auth/delivery/auth_handlers_test.go` — updated fake, added handler tests
- `internal/pkg/auth/repository/identity_storage.go` — added `DeleteByUserAndProvider` implementation
- `internal/pkg/auth/repository/identity_queries.go` — added `DeleteIdentityByUserAndProviderQuery`
- `internal/pkg/apperrors/auth.go` — added `IdentityAlreadyLinkedError`, `LastIdentityError`
- `internal/pkg/server/delivery/responses/responses.go` — added `ErrIdentityAlreadyLinked`, `ErrLastIdentity`
- `internal/pkg/server/delivery/routers/auth.go` — added routes
- `internal/pkg/server/delivery/routers/router.go` — pass `ctxUserKey` to auth handler
- `internal/pkg/auth/mocks/mock_auth.go` — regenerated

## Behavior / API changes
- `GET /api/auth/identities` — returns `[]UserIdentity` (login required)
- `POST /api/auth/link/{provider}` — links a new OAuth provider (login required, body: `LoginRequest`)
- `DELETE /api/auth/unlink/{provider}` — unlinks a provider (login required, must have >= 2 identities)
- Error 400 `Identity already linked to another user` if linking an identity owned by another user
- Error 400 `Cannot unlink last identity` if unlinking the last remaining identity
- Link/Unlink return 204 No Content on success

## Tests
- Usecase tests: `TestListIdentities` (happy + error), `TestLinkIdentity` (unsupported provider, auth error, already linked to other user, already linked to same user, create error, happy path), `TestUnlinkIdentity` (list error, last identity, delete error, happy path)
- Handler tests: `TestListIdentities` (happy + error), `TestLinkIdentity` (bad JSON, unsupported provider, already linked, happy path), `TestUnlinkIdentity` (last identity, not found, happy path)

## Verification
```
go build ./... -> OK
go test ./...  -> all pass
gofmt          -> clean
```

Standard verification: `go build ./... && go test ./... && gofmt -l ./internal/...`

## Decisions
- `LinkIdentity` is idempotent — if the identity is already linked to the same user, returns success (no-op)
- `UnlinkIdentity` checks identity count before deleting to prevent orphaned users
- `AuthHandler` now takes `ctxUserKey` parameter (internal constructor change, not API-facing)
- Link/Unlink return 204 No Content on success (no body)

## Rollout notes / Risks
- All three endpoints require authentication (behind `LoginRequiredMiddleware`)
- Frontend must implement UI for managing linked providers to expose this functionality
- `DELETE /api/auth/unlink/{provider}` returns 400 if user has only one identity

## Backlog updates
- Frontend needs new pages/components for identity management

---

# Audit Index

| Stage | Commit | Migrations | Breaking | Public API | Frontend required | Notes |
|-------|--------|------------|----------|------------|-------------------|-------|
| PR1+2 | `aaa7c42` | 007 | no | no | no | Cookie flags + timestamps |
| PR3a | `98720f1` | 008 | no (API additive) | yes | yes (USER_INACTIVE) | **Hard boundary**: DB column rename; binary + migration must deploy together |
| PR3b | `bc6b44e` | — | **yes** | yes | **yes** | JSON rename: `name`->`displayName`, `avatar`->`avatarUrl` |
| PR4 | `0189cab` | 009 | no | no | no | user_identity table + VK backfill |
| PR5 | `a1b4ac4` | 010 | no | no | no | DROP COLUMN vkid |
| PR6 | `79cf0dc` | — | **yes** | yes | **yes** | Route: `/login`->`/login/{provider}`; Redis mass logout |
| PR7 | `0d26c23` | — | no | yes | no | Google OAuth (additive) |
| PR8 | `c6880d4` | — | no | yes | no | Yandex OAuth (additive) |
| PR9 | `580a522` | — | no | yes | no | Identity management endpoints (additive) |

---

# Breaking Changes Summary

## 1. DB column rename (PR3a — `98720f1`, migration 008)

- **What:** `ALTER TABLE public."user" RENAME COLUMN name TO display_name; RENAME COLUMN avatar TO avatar_url`
- **Impact:** Old binary's SQL queries reference `name`/`avatar` columns — will get runtime SQL errors after migration
- **Deploy:** Migration 008 and PR3a binary must be deployed atomically
- **Rollback:** `go run cmd/app/main.go -migrate 7` + deploy pre-PR3a binary

## 2. JSON field renames (PR3b — `bc6b44e`)

- **What:** JSON response fields `name` -> `displayName`, `avatar` -> `avatarUrl`
- **Affected endpoints:** All responses embedding `User` object:
  - `POST /api/auth/login/{provider}` (response: `AuthResponse.User`)
  - `GET /api/auth/check` (response: `AuthResponse.User`)
  - WebSocket table messages containing `User`
- **Impact:** Frontend reads of `user.name` / `user.avatar` return `undefined`
- **Deploy:** Frontend and backend should deploy together
- **Rollback:** Revert binary to pre-PR3b; no migration needed

## 3. Auth route change (PR6 — `79cf0dc`)

- **What:** `POST /api/auth/login` -> `POST /api/auth/login/{provider}`
- **Impact:** Old frontend POSTing to `/api/auth/login` gets 404 (or 405)
- **Deploy:** Frontend and backend must deploy together
- **Rollback:** Revert binary to pre-PR6

## 4. Redis session format change (PR6 — `79cf0dc`)

- **What:** `FullSessionData` JSON now includes `"provider"` field. Old sessions serialized without this field become unreadable or parse with empty provider.
- **Impact:** **All existing user sessions invalidated.** `GET /api/auth/check` returns `{isAuth: false}` for all users. Users must log in again.
- **User-visible effect:** Forced re-login for all active users after deploy.
- **Deploy:** No special action needed — sessions self-heal on next login
- **Rollback:** Reverting binary means new-format sessions (created after deploy) won't parse. `FLUSHDB` on Redis session DB is the nuclear option; otherwise sessions expire naturally (30-day TTL).

## 5. vkid removal from JSON (PR5 — `a1b4ac4`, migration 010)

- **What:** `vkid` field no longer present in `User` JSON (Go field removed). DB column dropped in migration 010.
- **Impact:** If frontend ever read `user.vkID` — it returns `undefined`. Use `GET /api/auth/identities` instead.
- **Note:** The `vkid` JSON field was renamed to `vkID` (but still present) in earlier code. It was effectively invisible to most frontend code because it was rarely used. After PR5 it is fully gone.
- **Rollback:** `go run cmd/app/main.go -migrate 9` restores the column from user_identity data. **Fails if VK identity rows were deleted.**

---

# Rollback / Recovery Cheatsheet

## Checking current migration version

`golang-migrate` stores the version in the `schema_migrations` table in PostgreSQL:

```sql
SELECT version, dirty FROM schema_migrations;
```

If `dirty = true`, the last migration failed mid-way and must be resolved manually.

## Applying / rolling back migrations

```bash
# Apply all pending migrations
go run cmd/app/main.go -migrate latest

# Migrate to a specific version (applies up or down as needed)
go run cmd/app/main.go -migrate <version>
```

Version numbers correspond to migration file prefixes: `007`, `008`, `009`, `010`.

## Migration file reference

| Version | Name | Up | Down |
|---------|------|----|------|
| 007 | user_timestamps | ADD created_at, updated_at, last_login_at | DROP those columns |
| 008 | user_profile_status | RENAME name->display_name, avatar->avatar_url; ADD status, role, email, email_verified | DROP new cols; RENAME back |
| 009 | user_identity | CREATE TABLE user_identity; backfill from vkid; ALTER vkid DROP NOT NULL | Restore vkid NOT NULL; DROP TABLE |
| 010 | drop_vkid | DROP COLUMN vkid | Restore vkid from user_identity |

## Hard boundaries (rollback order matters)

### Rolling back past migration 008 (column rename)

1. Deploy pre-PR3a binary (or stop server)
2. `go run cmd/app/main.go -migrate 7`
3. Start pre-PR3a binary

If you run the old binary against the already-migrated DB (with `display_name`/`avatar_url` columns), SQL queries will fail.

### Rolling back past migration 010 (DROP vkid)

1. `go run cmd/app/main.go -migrate 9` — this restores `vkid` column from `user_identity` data
2. **Prerequisite:** `user_identity` table must still have VK provider rows. If identity rows were deleted (e.g., via `DELETE /api/auth/unlink/vk`), the down migration will set vkid to NULL for those users, and the subsequent `SET NOT NULL` will fail.
3. Deploy pre-PR5 binary

### Rolling back past migration 009 (user_identity table)

1. `go run cmd/app/main.go -migrate 8` — drops `user_identity` table, restores vkid NOT NULL
2. Deploy pre-PR4 binary
3. **Warning:** Any users created via Google/Yandex OAuth (no VK identity) will have NULL vkid and the down migration's `SET NOT NULL` will fail. Manual data cleanup required.

### Rolling back Redis session format (PR6)

No migration involved. Options:
- Accept that sessions created after PR6 deploy won't parse for old binary (they expire in 30 days)
- `FLUSHDB` on the Redis session database to force all users to re-login
- `redis-cli -n <db> KEYS "session:*" | xargs redis-cli -n <db> DEL` for targeted cleanup (verify key pattern first)
