# Frontend API Field Changes Backlog

Tracks all breaking and notable changes to JSON API responses that require frontend updates.
Updated as part of the [user-model-rollout-plan.md](user-model-rollout-plan.md).

---

## Field renames / removals

| Old field | New field | Struct / DTO | Affected endpoints | Change type | Status | Stage | Commit | Notes |
|-----------|-----------|--------------|--------------------|-------------|--------|-------|--------|-------|
| `name` | `displayName` | `User` | All responses embedding `User`: `POST /api/auth/login/{provider}`, `GET /api/auth/check`, WebSocket table messages | Rename | Done | PR3b | `bc6b44e` | **Breaking.** Frontend must replace all `user.name` reads with `user.displayName`. Logout response only contains `{isAuth: false}` — no User object, unaffected. |
| `avatar` | `avatarUrl` | `User` | Same as above | Rename | Done | PR3b | `bc6b44e` | **Breaking.** Now nullable (`omitempty`) — frontend must handle missing avatar gracefully. |
| `vkID` | *(removed)* | `User` | Same as above | Removal | Done | PR5 | `a1b4ac4` | **Breaking.** Field no longer present in User JSON. If frontend displayed VK ID, use `GET /api/auth/identities` instead (PR9). |

**Frontend action:** Find-and-replace all references to `user.name` -> `user.displayName` and `user.avatar` -> `user.avatarUrl`. Remove any reads of `user.vkID`. Handle `avatarUrl` being `undefined`/missing (show default avatar).

---

## Auth route changes

| Old route | New route | Change type | Status | Stage | Commit | Notes |
|-----------|-----------|-------------|--------|-------|--------|-------|
| `POST /api/auth/login` | `POST /api/auth/login/{provider}` | Route change | Done | PR6 | `79cf0dc` | **Breaking.** The `{provider}` is a URL path segment: `vk`, `google`, or `yandex`. Request body (`LoginRequest`) is unchanged: `{code, state, codeVerifier, deviceID}`. The provider is **not** a body field. |

**Frontend action:** Update API client to construct the login URL dynamically: `/api/auth/login/vk`, `/api/auth/login/google`, `/api/auth/login/yandex`. Update any hardcoded paths in tests.

---

## Session invalidation (one-time)

| Event | Trigger | User-visible effect | Status | Stage | Commit | Notes |
|-------|---------|---------------------|--------|-------|--------|-------|
| Redis session format change | Deploy of PR6 binary | **All active users are force-logged-out.** `GET /api/auth/check` returns `{isAuth: false}` for all pre-existing sessions. | Done | PR6 | `79cf0dc` | One-time event. After deploy, `FullSessionData` in Redis includes a new `provider` field. Old sessions lack it and fail deserialization. Sessions self-heal on next login. |

**Frontend action:** No code changes needed. Frontend already handles `isAuth: false` by redirecting to login. Be aware that all users will need to re-login after this deploy.

---

## New error responses

| Error code | HTTP status | Affected endpoints | Change type | Status | Stage | Commit | Notes |
|------------|-------------|-------------------|-------------|--------|-------|--------|-------|
| `USER_INACTIVE` | 403 | All protected endpoints (behind `loginRequiredMiddleware`) | New behavior | Done | PR3a | `98720f1` | Returned when user's `status` is not `active` (e.g., `banned`, `deleted`). Response body: `{"status": "USER_INACTIVE"}`. Frontend must distinguish from 401 `User not authorized`. |
| `Identity already linked to another user` | 400 | `POST /api/auth/link/{provider}` | New error | Done | PR9 | `580a522` | Returned when the OAuth identity is already linked to a different user account. |
| `Cannot unlink last identity` | 400 | `DELETE /api/auth/unlink/{provider}` | New error | Done | PR9 | `580a522` | Returned when user tries to unlink their only remaining identity. |

**Frontend action for `USER_INACTIVE`:** Add a 403 handler that checks for `status === "USER_INACTIVE"`. Display an "account suspended" or "account deleted" screen instead of the normal 403 forbidden page. This is distinct from 401 (session expired -> redirect to login).

---

## New fields on existing responses

| Field | Struct / DTO | Affected endpoints | Change type | Status | Stage | Commit | Notes |
|-------|-------------|-------------------|-------------|--------|-------|--------|-------|
| `status` | `User` | All responses embedding `User`: `POST /api/auth/login/{provider}`, `GET /api/auth/check` | New field | Done | PR3a | `98720f1` | Additive. Present for new logins; may be empty for old sessions. Treat empty/missing as `"active"`. Possible values: `"active"`, `"banned"`, `"deleted"`. |

Note: The `provider` field is stored internally in the Redis session (`FullSessionData.Provider`) but is **not** included in the `AuthResponse` sent to the client. The frontend knows which provider was used from the login URL path it called.

---

## New endpoints

| Method | Path | Auth required | Request body | Success response | Error responses | Status | Stage | Commit | Notes |
|--------|------|---------------|--------------|------------------|-----------------|--------|-------|--------|-------|
| `POST` | `/api/auth/login/{provider}` | no | `LoginRequest`: `{code, state, codeVerifier, deviceID}` | 200: `{isAuth: true, user: User}` + `Set-Cookie: session_id` | 400 (bad JSON, unsupported provider), 500 (OAuth/internal error) | Done | PR6 | `79cf0dc` | `{provider}` = `vk` / `google` / `yandex`. Replaces old `POST /api/auth/login`. |
| `GET` | `/api/auth/identities` | yes | — | 200: `UserIdentity[]` (see schema below) | 401 (not auth), 403 (inactive), 500 | Done | PR9 | `580a522` | Returns all linked identities for the current user. |
| `POST` | `/api/auth/link/{provider}` | yes | `LoginRequest`: `{code, state, codeVerifier, deviceID}` | 204 No Content | 400 (bad JSON, unsupported provider, already linked), 401, 403, 500 | Done | PR9 | `580a522` | Links a new OAuth provider to the current user. Idempotent if same user already linked. |
| `DELETE` | `/api/auth/unlink/{provider}` | yes | — | 204 No Content | 400 (last identity, not found), 401, 403, 500 | Done | PR9 | `580a522` | Unlinks a provider. Blocked if it's the user's only remaining identity. |

### `UserIdentity` response schema (`GET /api/auth/identities`)

```json
[
  {
    "id": 1,
    "userId": 42,
    "provider": "vk",
    "providerUserId": "12345",
    "email": "user@example.com",
    "createdAt": "2026-01-15T12:00:00Z"
  },
  {
    "id": 2,
    "userId": 42,
    "provider": "google",
    "providerUserId": "google-uid-abc",
    "email": "user@gmail.com",
    "createdAt": "2026-02-01T10:00:00Z"
  }
]
```

Fields: `id` (int), `userId` (int), `provider` (string), `providerUserId` (string), `email` (string, omitempty), `createdAt` (string, omitempty), `lastUsedAt` (string, omitempty).

Source: `internal/models/auth.go` — `UserIdentity` struct.

---

## Status legend

| Status | Meaning |
|--------|---------|
| **Planned** | Described in rollout plan, not yet implemented |
| **In Progress** | Backend PR is open or merged, frontend not yet updated |
| **Done** | Backend implemented and committed; frontend update pending or complete |

## Change type legend

| Change type | Meaning |
|-------------|---------|
| **Rename** | Existing JSON field name changed |
| **Removal** | Existing JSON field removed |
| **New field** | New field added to existing response |
| **New endpoint** | Entirely new API endpoint |
| **New behavior** | Existing endpoint returns a new error code or status in certain conditions |
| **New error** | New error response from a specific endpoint |
| **Route change** | Existing endpoint URL changed |

---

## Frontend Implementation Order

Recommended order for frontend changes:

1. **Update auth login route** (PR6 — mandatory, blocking)
   - Change `POST /api/auth/login` to `POST /api/auth/login/vk` (or dynamic `/{provider}`)
   - This must be deployed together with the backend PR6 change

2. **Update JSON field mapping** (PR3b — mandatory, blocking)
   - Replace `user.name` -> `user.displayName` everywhere
   - Replace `user.avatar` -> `user.avatarUrl` everywhere
   - Handle missing `avatarUrl` (show default avatar placeholder)
   - Remove any reads of `user.vkID`

3. **Handle `USER_INACTIVE` (403)** (PR3a — recommended)
   - Add 403 response handler for `{"status": "USER_INACTIVE"}`
   - Show "account suspended/deleted" UI, distinct from 401 unauthorized
   - This applies to all protected routes

4. **Handle forced re-login** (PR6 — awareness only)
   - No code change needed — existing `isAuth: false` handling covers it
   - Communicate to users that they may need to re-login after deploy

5. **Add Google/Yandex login buttons** (PR7/PR8 — optional, when ready)
   - Implement OAuth client-side flows for Google and Yandex
   - Call `POST /api/auth/login/google` or `POST /api/auth/login/yandex` with the authorization code

6. **Implement identity management UI** (PR9 — optional, when ready)
   - List linked providers: `GET /api/auth/identities`
   - Link new provider: `POST /api/auth/link/{provider}` (with OAuth code)
   - Unlink provider: `DELETE /api/auth/unlink/{provider}`
   - Handle error: "Cannot unlink last identity" (disable unlink button when only 1 identity)
   - Handle error: "Identity already linked to another user" (show user-friendly message)
