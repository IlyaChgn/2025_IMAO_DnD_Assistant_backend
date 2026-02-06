# User Model — Rollout Plan

> Companion document to [user-model-investigation-and-strategy-report.md](user-model-investigation-and-strategy-report.md).
> This plan covers **what** we change, **why**, and **in what order**.
> No code changes are made in this document — only the plan.

---

## A. Summary & Goals

### Why

The current `public.user` table has 4 columns (`id`, `vkid`, `name`, `avatar`).
VK is hardcoded as the only OAuth provider at every layer: schema, repository queries,
usecase logic, and API responses. Adding Google or Yandex would require touching all
of these in a single "big bang" — risky and untestable.

### What we unlock

1. **Multi-provider OAuth** — VK (existing), Google, Yandex.
2. **User metadata** — registration date, last login, status, role.
3. **Account management** — ban/soft-delete, admin flag.
4. **Foundation for future** — email notifications, analytics cohorts.

### Security principles (fixed decisions)

| Decision | Rationale |
|----------|-----------|
| **No automatic account linking by email** | A malicious actor who controls an email on one provider could hijack an account on another. Linking is manual only — the user must be logged in and explicitly connect a second provider. |
| **No refresh-token storage in DB** | Tokens live in Redis with TTL. If DB persistence is needed later, use AES-256-GCM encryption at application level. |
| **No phone storage** | VK provides phone — we continue ignoring it. |
| **Cookie hardening** | Add `Secure`, `SameSite=Lax` flags. |

---

## B. Target Data Model

### `public.user` (extended)

| Column | Type | Default | Nullable | Constraints | Notes |
|--------|------|---------|----------|-------------|-------|
| `id` | BIGINT | IDENTITY | NO | PK | Keep existing |
| `display_name` | TEXT | — | NO | CHECK(length < 100) | Renamed from `name` |
| `avatar_url` | TEXT | — | YES | — | Renamed from `avatar`, made nullable |
| `email` | TEXT | — | YES | — | From any provider; normalized to lowercase+trim; not necessarily unique (see below) |
| `email_verified` | BOOLEAN | FALSE | NO | — | `true` only if at least one linked provider guarantees the email is verified (see email rules) |
| `role` | TEXT | `'user'` | NO | — | `user` / `admin` |
| `status` | TEXT | `'active'` | NO | — | `active` / `banned` / `deleted` |
| `created_at` | TIMESTAMPTZ | `now()` | NO | — | Registration timestamp |
| `updated_at` | TIMESTAMPTZ | `now()` | NO | — | Last profile sync |
| `last_login_at` | TIMESTAMPTZ | — | YES | — | Last successful login via any provider |

**10 columns.** `vkid` is removed (moves to `user_identity`).

**Email uniqueness decision:** `email` on `users` is **NOT UNIQUE**. Reason: two users
may have the same email from different providers and never link accounts. Uniqueness is
enforced per-provider in `user_identity(provider, email)` if needed, not globally.

### Email rules

| Rule | Detail |
|------|--------|
| **Normalization** | Always `strings.ToLower(strings.TrimSpace(email))` before storing. Applied both in `users.email` and `user_identity.email`. |
| **`email_verified` semantics** | Set to `true` on `users` only when at least one linked identity has a provider-confirmed verified email (Google `email_verified=true`, Yandex default verified, VK — treat as **unverified** unless VK API explicitly confirms). Never set `email_verified=true` based on the user's own claim. |
| **Primary email** | `users.email` is populated from the **first identity that provides a verified email**. It is updated only when the user explicitly changes it or links a new provider with a verified email and `users.email` is still NULL. There is no automatic overwrite of an existing primary email. |
| **No auto-linking by email** | Even if two identities share the same verified email, we do NOT automatically merge accounts. Linking is always manual (user must be logged in). This is a fixed security decision. |

### `public.user_identity` (new)

| Column | Type | Default | Nullable | Constraints | Notes |
|--------|------|---------|----------|-------------|-------|
| `id` | BIGINT | IDENTITY | NO | PK | |
| `user_id` | BIGINT | — | NO | FK → user(id) ON DELETE CASCADE | |
| `provider` | TEXT | — | NO | — | `'vk'`, `'google'`, `'yandex'` |
| `provider_user_id` | TEXT | — | NO | — | External ID from the provider |
| `email` | TEXT | — | YES | — | Email reported by this provider |
| `created_at` | TIMESTAMPTZ | `now()` | NO | — | When identity was linked |
| `last_used_at` | TIMESTAMPTZ | — | YES | — | Last login via this provider |

**Indexes:**

| Index | Columns | Type | Purpose |
|-------|---------|------|---------|
| `uq_identity_provider` | `(provider, provider_user_id)` | UNIQUE | One external account maps to one identity row |
| `uq_identity_user_provider` | `(user_id, provider)` | UNIQUE | One identity per provider per user |
| `idx_identity_user_id` | `(user_id)` | INDEX | Fast JOIN on user lookups |

**No token columns.** Tokens stay in Redis session with TTL. If refresh-token
persistence is needed, add `encrypted_refresh_token BYTEA` + rotation key in config.

### Account linking flow (manual only)

```
1. User is logged in (session exists, user_id known).
2. User clicks "Link Google account" → frontend opens Google OAuth popup.
3. Frontend sends authorization code to POST /api/auth/link with {provider: "google", code: "..."}.
4. Backend exchanges code → gets Google user info.
5. Backend checks user_identity for (provider="google", provider_user_id=...).
   - If already linked to ANOTHER user → reject with 409 Conflict.
   - If already linked to THIS user → no-op, return OK.
   - If not found → INSERT into user_identity for current user_id.
6. User can unlink via DELETE /api/auth/unlink/google (guard: can't unlink last identity).
```

No automatic linking. No email matching. The user explicitly decides.

---

## C. API / JSON Changes (Breaking)

### Strategy: immediate breaking change + frontend backlog

We do NOT use a dual-output compatibility period. Reasons:

1. The project is pre-production / early stage — there is no public API contract.
2. Frontend and backend are developed by the same team.
3. Dual fields (`name` + `displayName`) add complexity for zero benefit.
4. Cleaner to coordinate a single breaking deployment.

**Process:** each breaking PR gets an entry in
[frontend-api-field-changes-backlog.md](frontend-api-field-changes-backlog.md).
Frontend updates are done in parallel or immediately after backend merge.

### Changed fields

| Old JSON field | New JSON field | Struct | Endpoints affected | PR |
|----------------|---------------|--------|-------------------|-----|
| `name` | `displayName` | `User` | `/api/auth/login`, `/api/auth/check`, all responses embedding `User` | PR 3b |
| `avatar` | `avatarUrl` | `User` | Same as above | PR 3b |
| `vkID` | *(removed from User)* | `User` | Same as above | PR 5 |
| *(new)* | `provider` | `AuthResponse` | `/api/auth/login`, `/api/auth/check` | PR 6 |
| *(new)* | `identities` | new endpoint | `GET /api/auth/identities` | PR 9 |

### New endpoints

| Method | Path | PR | Notes |
|--------|------|----|-------|
| GET | `/api/auth/identities` | PR 9 | List linked providers for current user |
| POST | `/api/auth/link` | PR 9 | Link new provider (must be logged in) |
| DELETE | `/api/auth/unlink/{provider}` | PR 9 | Unlink provider (can't unlink last) |

---

## D. Rollout Plan by PR

### PR 1 — Security quick wins (cookie flags)

**Goal:** Harden session cookie before any model changes.

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Delivery | `internal/pkg/auth/delivery/session.go` | Add `Secure` and `SameSite` flags to cookie (see rules below) |
| Config | `internal/pkg/config/config.go` + `config.yaml` | Add `session_duration` field (move 30-day constant from handler to config) |
| Delivery | `internal/pkg/auth/delivery/auth_handlers.go` | Read session duration from config instead of hardcoded const |

**Cookie flags by environment:**

| Flag | Production (`cfg.IsProd == true`) | Local dev (`cfg.IsProd == false`) |
|------|-----------------------------------|----------------------------------|
| `Secure` | `true` | `false` |
| `SameSite` | `http.SameSiteLaxMode` | `http.SameSiteLaxMode` |
| `HttpOnly` | `true` (already set) | `true` (already set) |

- **`Secure` depends on environment** — in prod, cookie is only sent over HTTPS. In local dev,
  `Secure: false` so that `http://localhost` keeps working without certificates.
  The handler reads `cfg.IsProd` and sets the flag accordingly.
- **`SameSite=Lax`** is the safe default: cookie is sent on top-level navigations but NOT on
  cross-site subrequests (protects against CSRF).

**SameSite decision checklist (Lax vs None):**

The correct `SameSite` value depends on the deployment topology, which may differ between
local dev, staging, and production:

| Question | If YES → | If NO → |
|----------|----------|---------|
| Are frontend and API on the **same registrable domain** (e.g. `app.example.com` + `api.example.com`, or both on `localhost`)? | `SameSite=Lax` is safe. | Cookie may not be sent on OAuth redirect back — see next row. |
| After OAuth redirect from Google/Yandex, does the browser send the `session_id` cookie back to the API? (Check DevTools → Application → Cookies on the redirect response.) | `SameSite=Lax` works — keep it. | Switch to `SameSite=None` + `Secure=true` for that environment. |
| Is `SameSite=None` required? | Make `SameSite` **configurable via config** (env var or yaml), defaulting to `Lax`. Set `None` only where needed. `None` requires `Secure=true` — browsers reject `None` without `Secure`. | Keep `Lax`. |

**When to verify:** Run this checklist manually during PR 7 (Google) and PR 8 (Yandex) — these
are the first PRs that exercise a non-VK OAuth redirect. VK may also need verification if
the deployment topology changes.

**Migration:** None.

**Risks:** Minimal. `Secure` gated by `cfg.IsProd` ensures local development is not affected.
`SameSite` defaults to `Lax`; if OAuth redirects fail in a particular environment, the
checklist above provides the resolution path.

**Testing:**
- Unit: handler test asserts cookie flags are set correctly for both `isProd=true` and `isProd=false`.
- No integration test needed.

---

### PR 2 — Add timestamps to `users`

**Goal:** Add `created_at`, `updated_at`, `last_login_at` to track user lifecycle.

**Migration** (`007_user_timestamps.up.sql`):
```sql
ALTER TABLE public.user
    ADD COLUMN created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN last_login_at TIMESTAMPTZ;
```

Existing rows get `created_at = now()` — approximate but acceptable (no historical data).

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Model | `internal/models/auth.go` | Add `CreatedAt`, `UpdatedAt`, `LastLoginAt` fields with `json:"-"` |
| Repository | `internal/pkg/auth/repository/auth_queries.go` | Update `CreateUserQuery` to include `created_at`, `updated_at`; update `UpdateUserQuery` to SET `updated_at = now()` |
| Repository | `internal/pkg/auth/repository/auth_storage.go` | Scan new columns in query results |
| Usecase | `internal/pkg/auth/usecases/auth.go` | After successful session creation, update `last_login_at` (best-effort — see rule below) |

**`last_login_at` update rule:**

- Updated **after** the session is successfully created in Redis (i.e. the login is already
  successful from the user's perspective).
- **Best-effort:** if the `UPDATE ... SET last_login_at = now()` query fails, the login
  still succeeds. The error is logged (+ metric increment if available) but NOT propagated
  to the caller. The user's session is valid regardless.
- Rationale: `last_login_at` is an analytics/audit field. A transient DB hiccup should never
  block a user from logging in.

**API impact:** None. New fields are `json:"-"`.

**Risks:** Minimal. Additive ALTER, no NOT NULL without default, no column renames.

**Testing:**
- Unit (usecase): auth usecase tests verify `last_login_at` update is called after session creation (mock repo expectation via gomock).
- Unit (usecase): verify that if `last_login_at` update fails, `Login()` still returns success (error is swallowed).
- Integration (`//go:build integration`): apply migration to test DB, insert user, verify timestamp defaults. Repository queries tested against real Postgres.

---

### PR 3a — Schema changes + internal Go model (no API break)

**Goal:** Rename DB columns, add new columns, update Go structs and repository queries.
The JSON API **stays unchanged** in this PR — the model temporarily keeps the old JSON tags
(`json:"name"`, `json:"avatar"`) so the frontend is not affected.

**Migration** (`008_user_fields.up.sql`):
```sql
ALTER TABLE public.user RENAME COLUMN name   TO display_name;
ALTER TABLE public.user RENAME COLUMN avatar TO avatar_url;
ALTER TABLE public.user ALTER COLUMN avatar_url DROP NOT NULL;
ALTER TABLE public.user DROP CONSTRAINT IF EXISTS "user_avatar_check";
ALTER TABLE public.user
    ADD COLUMN email          TEXT,
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN status         TEXT    NOT NULL DEFAULT 'active',
    ADD COLUMN role           TEXT    NOT NULL DEFAULT 'user';
```

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Model | `internal/models/auth.go` | Rename Go fields: `Name` → `DisplayName`, `Avatar` → `AvatarURL`. **Keep old JSON tags** for now: `json:"name"`, `json:"avatar"`. Add `Email`, `EmailVerified`, `Status`, `Role` (all `json:"-"`). |
| Repository | `auth/repository/auth_queries.go` | All queries: `name` → `display_name`, `avatar` → `avatar_url`. Apply `lower(trim(...))` on email before INSERT/UPDATE. |
| Repository | `auth/repository/auth_storage.go` | Update Scan calls for renamed columns. |
| Usecase | `auth/usecases/auth.go` | Use `user.DisplayName`, `user.AvatarURL` internally; save normalized email from VK if available. |
| Middleware | `middleware/auth/login_required.go` | After CheckAuth: if `user.Status != "active"`, return 403 (see error contract below). |
| Table | `internal/pkg/table/` | Update references from `user.Name` to `user.DisplayName` in Go code. |

**Inactive user error contract:**

When the auth middleware detects `user.Status != "active"` (i.e. `banned` or `deleted`):

| Aspect | Value |
|--------|-------|
| HTTP status | `403 Forbidden` |
| Error code (response body) | `"USER_INACTIVE"` |
| Response format | `{"status": "USER_INACTIVE"}` (same shape as existing error responses) |
| Where implemented | `middleware/auth/login_required.go` — after successful session lookup, before injecting user into context |
| Frontend handling | Frontend receives 403 with `USER_INACTIVE` instead of the usual 401 `NOT_AUTHORIZED`. Should display "account suspended" or redirect to a support/info page. |

We use a single `USER_INACTIVE` code for both `banned` and `deleted` — the user does not
need to know the internal reason. 403 (not 401) because the user IS authenticated (valid
session) but is NOT authorized to use the service.

**Implementation note:** the error must be sent using the existing shared helper, exactly as
all other errors in the project:

```go
responses.SendErrResponse(w, responses.StatusForbidden, responses.ErrUserInactive)
```

Add a new constant `ErrUserInactive = "USER_INACTIVE"` to
`internal/pkg/server/delivery/responses/responses.go` alongside existing constants
(`ErrNotAuthorized`, `ErrForbidden`, etc.). This produces the standard response shape
`{"status": "USER_INACTIVE"}` via `newErrResponse()`. Do NOT construct JSON manually
in the middleware — use `SendErrResponse` to keep the format consistent project-wide.

**API impact:** New error response `USER_INACTIVE` on all protected endpoints. JSON payload shape
is unchanged (`{"status": "..."}`) — but this is a new status code that the frontend must handle.
See [frontend-api-field-changes-backlog.md](frontend-api-field-changes-backlog.md).

**Risks:**
- Column rename can break concurrent queries during deploy. Mitigation: deploy during low
  traffic or maintenance window (early-stage project).
- `DROP CONSTRAINT` on avatar CHECK — must match exact constraint name. `IF EXISTS` handles this.

**Testing:**
- Unit: usecase tests verify email is normalized and saved when VK provides it.
- Unit: dedicated middleware test for `status=banned` → 403 + `USER_INACTIVE` body.
  Cover this with a standalone `LoginRequiredMiddleware` unit test that injects a mock
  usecase returning a user with `Status="banned"`, then asserts 403 + `ErrUserInactive`
  payload via `responses.SendErrResponse`. Handler contract tests do not need to duplicate
  this — the middleware is the single enforcement point.
- Unit: middleware test for `status=active` → passes through (existing behavior).
- Unit: handler tests still pass with old JSON field names (no change in delivery).
- Integration: apply migration on DB with existing data, verify column rename + defaults.

---

### PR 3b — API/JSON breaking changes (`name` → `displayName`, `avatar` → `avatarUrl`)

**Goal:** Update JSON tags to the new field names. This is the **first breaking API change**.
Separated from PR 3a so that the DB migration and the API break are independent deployments —
if the migration causes issues, the API is still intact.

**Migration:** None (schema already updated in PR 3a).

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Model | `internal/models/auth.go` | Change JSON tags: `json:"name"` → `json:"displayName"`, `json:"avatar"` → `json:"avatarUrl,omitempty"` |
| Delivery | All handlers referencing old JSON field names in tests/assertions | Update to new names |
| Table | `internal/pkg/table/` | If `Participant` has its own JSON tags for name/avatar — update those too |

**API impact:** **BREAKING.** See [frontend-api-field-changes-backlog.md](frontend-api-field-changes-backlog.md).
Frontend must update all reads of `user.name` → `user.displayName` and `user.avatar` → `user.avatarUrl`.

**Risks:** Minimal — code change is only JSON tag strings. Coordinate merge with frontend deployment.

**Testing:**
- Unit: all auth handler tests updated for new JSON field names.
- Unit: verify JSON serialization output matches `displayName` / `avatarUrl`.

---

### PR 4 — Create `user_identity` table, migrate VK data

**Goal:** Introduce `user_identity` table. Copy existing VK identities from `user.vkid`. Keep `vkid` column temporarily (nullable) to allow rollback.

**Migration** (`009_user_identity.up.sql`):
```sql
CREATE TABLE IF NOT EXISTS public.user_identity (
    id               BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES public.user(id) ON DELETE CASCADE,
    provider         TEXT   NOT NULL,
    provider_user_id TEXT   NOT NULL,
    email            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at     TIMESTAMPTZ,
    CONSTRAINT uq_identity_provider      UNIQUE (provider, provider_user_id),
    CONSTRAINT uq_identity_user_provider UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_identity_user_id ON public.user_identity(user_id);

-- Backfill: copy VK identities from user table (idempotent)
INSERT INTO public.user_identity (user_id, provider, provider_user_id, created_at)
SELECT id, 'vk', vkid, created_at
FROM public.user
WHERE vkid IS NOT NULL
ON CONFLICT (provider, provider_user_id) DO NOTHING;
```

The `ON CONFLICT DO NOTHING` makes the backfill idempotent — re-running the migration
(or running it after a partial failure) is safe and skips already-inserted rows.

**Do NOT drop `vkid` yet.** Make it nullable in the same migration:
```sql
ALTER TABLE public.user ALTER COLUMN vkid DROP NOT NULL;
```

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Model | `internal/models/auth.go` | Add `UserIdentity` struct |
| Interfaces | `internal/pkg/auth/interfaces.go` | Add `IdentityRepository` interface: `FindByProvider(ctx, provider, providerUserID) (*UserIdentity, error)`, `CreateIdentity(ctx, *UserIdentity) error`, `UpdateLastUsed(ctx, identityID int) error`, `ListByUserID(ctx, userID int) ([]UserIdentity, error)` |
| Repository | New file `auth/repository/identity_storage.go` | Implement `IdentityRepository` against `user_identity` table |
| Repository | `auth/repository/auth_queries.go` | `CheckUser` query changes: lookup via `user_identity` JOIN instead of `WHERE vkid = $1` |
| Usecase | `auth/usecases/auth.go` | Login: `identityRepo.FindByProvider("vk", vkUserID)` → if found, get user; if not, create user + identity |
| Wiring | `internal/pkg/server/app.go` | Create `IdentityRepository`, inject into auth usecase |

**`models.User` still has `VKID` field** — set from identity lookup, used only for backward compat in this PR. Will be removed in PR 5.

**Risks:**
- Data migration (`INSERT INTO ... SELECT`) must be idempotent. Handled by `ON CONFLICT DO NOTHING` in the backfill query.
- Rollback: `down.sql` drops `user_identity` table, restores `vkid NOT NULL`.

**Testing:**
- Unit (usecase level): auth usecase tests updated — mock both `AuthRepository` and `IdentityRepository` via gomock.
- Integration (`//go:build integration`): `IdentityRepository` methods tested against real Postgres — verify INSERT, FindByProvider, UpdateLastUsed, ListByUserID with actual SQL.
- Integration: apply migration on DB with existing users, verify all have corresponding identity rows (backfill correctness).
- Integration: login flow creates identity row for new users.

> **Why no unit tests for repository layer?** Mocking `pgxpool` at the SQL level gives false
> confidence — the mock verifies the query string, not that the query actually works against
> Postgres. Repository/adapter tests belong in integration tests with a real DB, as established
> in [docs/TESTING.md](../docs/TESTING.md). Unit tests stay at the usecase level where gomock
> provides real value (mocking repository interfaces, not SQL drivers).

---

### PR 5 — Drop `vkid` from `users`, remove `VKID` from Go model

**Goal:** Clean up. Remove the transitional `vkid` column and the `VKID` field from `models.User`.

**Migration** (`010_drop_vkid.up.sql`):
```sql
ALTER TABLE public.user DROP COLUMN IF EXISTS vkid;
```

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Model | `internal/models/auth.go` | Remove `VKID string` field from `User` struct |
| Repository | `auth/repository/auth_queries.go` | Remove any remaining `vkid` references in SELECT/INSERT |
| Delivery | `auth/delivery/auth_handlers.go` | `AuthResponse` no longer includes `vkID` in JSON |

**API impact:** **BREAKING** — `vkID` field removed from User JSON. See backlog.

**Risks:** Low — by this point, all code reads from `user_identity`. The column is already nullable and unused.

**Testing:**
- Unit: verify `User` JSON serialization has no `vkID` field.
- All existing auth tests already work against identity-based flow from PR 4.

---

### PR 6 — Abstract `OAuthProvider` interface

**Goal:** Replace VK-specific `VKApi` interface with provider-agnostic `OAuthProvider`. Refactor auth usecase to accept any provider.

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Interfaces | `internal/pkg/auth/interfaces.go` | New interface `OAuthProvider`: `ExchangeCode(ctx, *OAuthCodeRequest) (*OAuthTokens, error)`, `GetUserInfo(ctx, *OAuthTokens) (*OAuthUserInfo, error)`, `ProviderName() string` |
| Model | `internal/models/auth.go` | New structs: `OAuthCodeRequest` (code, state, codeVerifier, redirectURI), `OAuthTokens` (accessToken, refreshToken, idToken), `OAuthUserInfo` (providerUserID, displayName, avatarURL, email, emailVerified) |
| External | `internal/pkg/auth/external/vk_api.go` | Implement `OAuthProvider` interface (adapter around existing VK logic) |
| Usecase | `auth/usecases/auth.go` | `authUsecases` holds `map[string]OAuthProvider` instead of single `VKApi`. `Login()` receives `provider` string param, selects provider from map. |
| Delivery | `auth/delivery/auth_handlers.go` | `LoginRequest` gains `provider` field (default `"vk"` if empty for backward compat). Pass to usecase. |
| Delivery | `auth/delivery/auth_handlers.go` | `AuthResponse` gains `provider` field (which provider was used for this session). |
| Wiring | `internal/pkg/server/app.go` | Build `map[string]OAuthProvider{"vk": vkProvider}`, inject into usecase |
| Session | `internal/models/auth.go` | `FullSessionData.Tokens` becomes provider-agnostic `OAuthTokens` |

**Migration:** None.

**API impact:** `LoginRequest` now accepts optional `provider` field. `AuthResponse` gains `provider` field. See backlog.

**Risks:**
- Large refactor of auth interfaces — all auth tests must be rewritten.
- `FullSessionData` format changes — existing Redis sessions become invalid. This is an
  **accepted one-time cost** for an early-stage project. User-visible behavior: `GET /api/auth/check`
  returns `{isAuth: false}` (the deserialization of the old format fails, `GetSession` returns
  `false`), and the frontend redirects to login as usual. No 500s, no crashes — just a forced
  re-login for all active sessions. Mitigation: deploy during low traffic.

**Testing:**
- Unit: all auth usecase tests rewritten against `MockOAuthProvider` (gomock).
- Unit: VK adapter tests — verify it correctly implements `OAuthProvider`.
- Unit: handler tests — verify `provider` field in request/response.
- Manual: VK login flow still works end-to-end.

---

### PR 7 — Add Google OAuth

**Goal:** Implement Google as second OAuth provider.

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| External | New file `auth/external/google_api.go` | Implement `OAuthProvider` for Google: exchange via `https://oauth2.googleapis.com/token`, user info via `https://www.googleapis.com/oauth2/v3/userinfo` |
| Config | `config.go` + `.env` | Add `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI` |
| Wiring | `internal/pkg/server/app.go` | Register `"google": googleProvider` in provider map |

**Migration:** None — `user_identity` already supports any provider string.

**Risks:**
- Google OAuth has different scopes/claims than VK. The `OAuthProvider` abstraction from PR 6 must handle this. Mitigation: `OAuthUserInfo` struct is provider-agnostic; each implementation maps provider-specific fields.
- **Cookie/SameSite check:** verify that the Google OAuth redirect flow works with the current
  cookie flags (`SameSite=Lax`, `Secure` gated by env) in a real browser. If frontend and
  backend are on different origins, `SameSite=Lax` may prevent the cookie from being sent on
  the redirect back from Google. Test this in staging before merging.

**Testing:**
- Unit: `GoogleOAuthProvider` tested with mock HTTP client (same pattern as VK).
- Unit: auth usecase test with `provider="google"` — verify user + identity creation.
- Manual: full Google login flow with real credentials. **Explicitly verify cookie is sent
  after OAuth redirect** (check browser DevTools → Application → Cookies).

---

### PR 8 — Add Yandex OAuth

**Goal:** Implement Yandex as third OAuth provider. Same pattern as PR 7.

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| External | New file `auth/external/yandex_api.go` | Implement `OAuthProvider` for Yandex: exchange via `https://oauth.yandex.ru/token`, user info via `https://login.yandex.ru/info` |
| Config | `config.go` + `.env` | Add `YANDEX_CLIENT_ID`, `YANDEX_CLIENT_SECRET`, `YANDEX_REDIRECT_URI` |
| Wiring | `internal/pkg/server/app.go` | Register `"yandex": yandexProvider` in provider map |

**Migration:** None.

**Risks:**
- Same cookie/SameSite concern as PR 7 — verify Yandex OAuth redirect flow works with
  `SameSite=Lax` in a real browser on staging.

**Testing:** Same pattern as PR 7 (including manual cookie verification after OAuth redirect).

---

### PR 9 — Manual linking / unlinking endpoints

**Goal:** Allow logged-in users to link additional providers and unlink existing ones.

**Changes:**

| Layer | File | Change |
|-------|------|--------|
| Interfaces | `auth/interfaces.go` | Add to `AuthUsecases`: `LinkProvider(ctx, userID int, provider string, code *OAuthCodeRequest) error`, `UnlinkProvider(ctx, userID int, provider string) error`, `ListIdentities(ctx, userID int) ([]UserIdentity, error)` |
| Usecase | `auth/usecases/auth.go` | `LinkProvider`: exchange code via OAuthProvider → check identity doesn't belong to another user → create identity row. `UnlinkProvider`: verify user has >1 identity → delete row. `ListIdentities`: delegate to IdentityRepository. |
| Delivery | New file `auth/delivery/identity_handlers.go` | `POST /api/auth/link` — requires login middleware. `DELETE /api/auth/unlink/{provider}` — requires login. `GET /api/auth/identities` — requires login. |
| Router | `server/delivery/routers/auth.go` | Register new routes behind `loginRequiredMiddleware` |

**Migration:** None.

**API:**
- `GET /api/auth/identities` → `[{provider: "vk", createdAt: "..."}, {provider: "google", ...}]`
- `POST /api/auth/link` body: `{provider: "google", code: "...", codeVerifier: "...", ...}`
- `DELETE /api/auth/unlink/google` → 200 OK or 400 "can't unlink last provider"

**Risks:**
- Race condition: two requests to link the same provider simultaneously. Mitigation: UNIQUE constraint on `(user_id, provider)` handles this at DB level — second INSERT fails.
- Linking a provider identity that belongs to another user → 409 Conflict response.

**Testing:**
- Unit: usecase tests for link (happy path, already-linked-to-other-user, duplicate).
- Unit: usecase tests for unlink (happy path, last-identity guard).
- Unit: handler tests for new endpoints (status codes, error bodies).
- Integration: full link/unlink cycle against test DB.

---

### PR 10 — Cleanup and deprecations (optional, deferred)

**Goal:** Remove any remaining legacy code/fields.

Candidates:
- Remove `VKApi` interface if fully replaced by `OAuthProvider`.
- Remove `VKTokensData`, `UserPublicInfo` structs if unused.
- Remove `LoginRequest.DeviceID` if Google/Yandex don't use it (or make it VK-specific).
- Audit `FullSessionData` — ensure it's provider-agnostic.

This PR is intentionally last and optional. Ship it when the dust settles.

---

## E. Deployment Notes

### Stop-the-world requirement for rename migrations

Migration 008 (PR 3a) uses `ALTER TABLE RENAME COLUMN` (`name` → `display_name`,
`avatar` → `avatar_url`). After this migration is applied, **any running instance
built from the old code will crash** — its SQL queries reference columns `name` and
`avatar` which no longer exist.

Our deploy model is already safe for this: we stop the old process, run migrations,
then start the new binary. There is no window where old code hits the new schema.
However, this must be a **hard rule** for any migration that renames or drops columns:

1. **No concurrent old instances/workers.** Verify that no background jobs, cron
   tasks, or secondary replicas are still running old code before applying the migration.
2. **No rollback to old binary without down-migration.** If the new binary fails to
   start after migration 008, you cannot simply restart the old binary — it will fail
   on missing columns. You must run the down-migration first (`migrate` to version 7),
   then start the old binary.

### Rollback rehearsal recommendation

Before applying rename/drop migrations to production, rehearse the full cycle on a
staging or local copy of the database:

- [ ] Apply migrations to latest (`-migrate latest`) — verify success
- [ ] Roll back to the last stable version (`-migrate <previous_version>`) — verify
  the down-migration restores the original column names
- [ ] Re-apply to latest (`-migrate latest`) — verify the up-migration is idempotent
  and succeeds again

This is especially important for migration 008 (RENAME COLUMN): if the down-migration
has a bug, you lose the ability to roll back on production. Catching this on staging
costs nothing; discovering it on production means extended downtime.

---

## F. Dependency Graph

```
PR 1   (cookie flags)          — independent
PR 2   (timestamps)            — independent
PR 3a  (schema + internal)     — independent
PR 3b  (JSON breaking)         — depends on PR 3a
PR 4   (user_identity table)   — depends on PR 2 (needs created_at for backfill)
PR 5   (drop vkid)             — depends on PR 4
PR 6   (OAuthProvider)         — depends on PR 4
PR 7   (Google)                — depends on PR 6
PR 8   (Yandex)                — depends on PR 6
PR 9   (link/unlink)           — depends on PR 6
PR 10  (cleanup)               — depends on all above
```

> **Note:** "independent" refers to development and merge order — these PRs have no code
> dependencies on each other and can be worked on in parallel. Database migrations must still
> be applied in sequential migration-number order during deployment (`007`, `008`, `009`, ...).

PRs 1, 2, 3a can be developed and merged in parallel.
PR 3b can merge right after PR 3a (coordinate with frontend).
PR 4 does NOT depend on PR 3a: the backfill reads `vkid` and `created_at` (from PR 2),
not `display_name`. PR 3a and PR 4 can be developed in parallel after PR 2 merges.
PRs 7, 8, 9 can be developed in parallel after PR 6 merges.

---

## G. Testing Strategy (cross-cutting)

| Test type | Where | What |
|-----------|-------|------|
| **Usecase unit tests** (gomock) | `auth/usecases/*_test.go` | Mock repos + mock OAuthProvider via gomock. Table-driven. Cover happy path, repo errors, provider errors, status checks. |
| **Handler contract tests** | `auth/delivery/*_test.go` | Hand-written fakes. Assert HTTP status codes, JSON field names, cookie flags. |
| **Repository integration tests** | `auth/repository/*_test.go` (`//go:build integration`) | Real Postgres. Verify actual SQL behavior: inserts, lookups, constraints, edge cases. |
| **Full-flow integration tests** | `auth/*_test.go` (`//go:build integration`) | Real Postgres + Redis. Apply migrations, run full login flow, verify DB state. |
| **Migration tests** | CI + local | Apply migrations to empty DB and to DB with existing data. Verify no errors, constraints hold. |

> **Repository testing philosophy:** We do NOT unit-test repositories with gomock for pgxpool.
> Mocking SQL drivers produces tests that verify query strings, not actual database behavior.
> A query can pass a mock test and still fail against real Postgres (wrong column name, type
> mismatch, missing index). Repositories are tested via integration tests with a real DB.
> Unit tests stay at the usecase level where mocking repository **interfaces** (not drivers)
> provides genuine coverage.

All tests follow existing conventions from [docs/TESTING.md](../docs/TESTING.md):
table-driven, `t.Parallel()`, gomock for usecases, hand-written fakes for delivery,
`errors.Is()` assertions, no `time.Sleep`.
