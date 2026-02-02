# Frontend API Field Changes Backlog

Tracks all breaking and notable changes to JSON API responses that require frontend updates.
Updated as part of the [User Model Rollout Plan](User%20Model%20-%20Rollout%20Plan.md).

---

## Field renames / removals

| Old field | New field | Struct / DTO | Affected endpoints | Change type | Status | PR | Frontend ticket/owner | Notes |
|-----------|-----------|--------------|--------------------|-------------|--------|----|----------------------|-------|
| `name` | `displayName` | `User` | `POST /api/auth/login`, `GET /api/auth/check`, all responses embedding `User` | Rename | Done | PR 3b | — | Breaking. All places where frontend reads `user.name` must switch to `user.displayName`. Logout response only contains `{isAuth: false}` — no User object. |
| `avatar` | `avatarUrl` | `User` | Same as above | Rename | Done | PR 3b | — | Breaking. Now nullable (`omitempty`) — frontend must handle missing avatar. |
| `vkID` | *(removed)* | `User` | Same as above | Removal | Planned | PR 5 | — | Breaking. Field no longer present in User JSON. If frontend displays VK ID, use `GET /api/auth/identities` instead. |

## New error responses

| Error code | HTTP status | Affected endpoints | Change type | Status | PR | Frontend ticket/owner | Notes |
|------------|-------------|-------------------|-------------|--------|----|----------------------|-------|
| `USER_INACTIVE` | 403 | All protected endpoints (behind `loginRequiredMiddleware`) | New behavior | Done | PR 3a | — | Returned when user's `status` is `banned` or `deleted`. Frontend should distinguish this from 401 `NOT_AUTHORIZED`: display "account suspended" message or redirect to info page. Response body: `{"status": "USER_INACTIVE"}`. |
| Session invalidation | — | `GET /api/auth/check` returns `{isAuth: false}` for all existing sessions | New behavior | Planned | PR 6 | — | One-time: deploying PR 6 changes the Redis session format. All active sessions become unreadable and `check` returns `isAuth: false`. Frontend handles this normally (redirect to login). No action needed beyond awareness. |

## New fields on existing responses

| Field | Struct / DTO | Affected endpoints | Change type | Status | PR | Frontend ticket/owner | Notes |
|-------|-------------|-------------------|-------------|--------|----|----------------------|-------|
| `status` | `User` | `POST /api/auth/login`, `GET /api/auth/check`, all responses embedding `User` | New field | Done | PR 3a | — | Additive field. Present for new logins; may be empty for old sessions; treat empty as `"active"`. Values: `"active"`, `"banned"`, `"deleted"`. |
| `provider` | `AuthResponse` | `POST /api/auth/login`, `GET /api/auth/check` | New field | Planned | PR 6 | — | String: `"vk"`, `"google"`, `"yandex"`. Indicates which provider was used for current session. |

## New endpoints

| Method | Path | Change type | Status | PR | Frontend ticket/owner | Notes |
|--------|------|-------------|--------|----|----------------------|-------|
| GET | `/api/auth/identities` | New endpoint | Planned | PR 9 | — | Returns list of linked providers: `[{provider, createdAt}]`. Requires auth. |
| POST | `/api/auth/link` | New endpoint | Planned | PR 9 | — | Link a new OAuth provider. Body: `{provider, code, codeVerifier, state}`. Requires auth. |
| DELETE | `/api/auth/unlink/{provider}` | New endpoint | Planned | PR 9 | — | Unlink a provider. Returns 400 if it's the last linked provider. Requires auth. |

## Changed request fields

| Field | Struct / DTO | Endpoint | Change type | Status | PR | Frontend ticket/owner | Notes |
|-------|-------------|----------|-------------|--------|----|----------------------|-------|
| `provider` | `LoginRequest` | `POST /api/auth/login` | New field (request) | Planned | PR 6 | — | New optional field. Default: `"vk"` (backward compatible). Set to `"google"` or `"yandex"` for other providers. |

## Status legend

| Status | Meaning |
|--------|---------|
| **Planned** | Described in rollout plan, not yet implemented |
| **In Progress** | Backend PR is open or merged, frontend not yet updated |
| **Done** | Both backend and frontend updated and deployed |

## Change type legend

| Change type | Meaning |
|-------------|---------|
| **Rename** | Existing JSON field name changed |
| **Removal** | Existing JSON field removed |
| **New field** | New field added to existing response |
| **New field (request)** | New field added to request body |
| **New endpoint** | Entirely new API endpoint |
| **New behavior** | Existing endpoint returns a new error code or status in certain conditions |
