# Auth Frontend Integration Guide

> **Source of truth:** This document is generated from the Go source code in `internal/pkg/auth/`, `internal/pkg/middleware/auth/`, `internal/pkg/server/`, and `internal/models/auth.go`.
> Every fact below was verified against the codebase on the `map-editor` branch (HEAD `e167413`).
> Where uncertainty exists, it is marked with **VERIFY:**.

---

# TL;DR Quickstart

Minimal steps to get auth working end-to-end:

1. **Configure HTTP client** — set base URL to `http://localhost:8080/api` (dev) and `credentials: "include"` on every request.
2. **Restore session on app load** — call `GET /api/auth/check`. If `isAuth === true`, use the returned `user`. If `false`, show login screen.
3. **Start OAuth flow** — open the provider's authorization page in a browser/popup (VK, Google, or Yandex). Generate a random `state`, save it in `sessionStorage`. For VK, also generate a `code_verifier` and derive the `code_challenge`.
4. **Handle OAuth redirect** — after the user authorizes, the provider redirects back with `?code=...&state=...`. Compare `state` against the saved value; abort if mismatch.
5. **Exchange code via backend** — `POST /api/auth/login/{provider}` with `{"code": "..."}` (+ `state`, `codeVerifier`, `deviceID` for VK). Backend sets the `session_id` cookie automatically.
6. **Use the login response** — read `isAuth` and `user` (`displayName`, `avatarUrl` which may be missing, `status`). There is no separate `/api/auth/me` endpoint; the login response is the user profile.
7. **Handle errors globally** — intercept `401` → redirect to login; intercept `403` with `status === "USER_INACTIVE"` → show "account suspended" screen; `400` → show field-level error; `500` → retry / generic error.
8. **Logout** — `POST /api/auth/logout`. Rely on `isAuth: false` in the response; ignore the `user` object. Clear local user state.
9. **Identity management (optional)** — list linked providers via `GET /api/auth/identities`, link new ones via `POST /api/auth/link/{provider}`, unlink via `DELETE /api/auth/unlink/{provider}`. Disable unlink when only 1 identity remains.

---

# Overview

The backend provides a multi-provider OAuth authentication system with session-based auth using cookies. Supported providers: **VK**, **Google**, **Yandex**.

Key characteristics:
- Sessions are stored in **Redis** with a configurable TTL (default 720h / 30 days).
- The session cookie (`session_id`) is set on login and cleared on logout.
- Protected endpoints go through `LoginRequiredMiddleware` which validates the session and checks user status.
- The `provider` is **always** a URL path variable (`{provider}`), **never** a request body field.
- All error responses use the envelope `{"status": "<error string>"}`.

---

# Base URLs and assumptions

| Environment | Base URL | Notes |
|-------------|----------|-------|
| Development | `http://localhost:8080/api` | CORS allows `http://localhost:3000` (from `config.yaml`) |
| Production | Depends on deploy | Passed via `-prod` flag; cookie `Secure` flag is `true` |

All auth endpoints are under `/api/auth/`.

CORS is configured via `gorilla/handlers.CORS` in `internal/pkg/server/app.go:265`:
- `AllowCredentials()` — sends `Access-Control-Allow-Credentials: true`
- `AllowedOrigins` — from `cfg.Server.Origins` (default: `["http://localhost:3000"]`)
- `AllowedHeaders` — `["X-Requested-With", "Content-Type"]`
- `AllowedMethods` — `["GET", "POST", "DELETE", "PUT", "HEAD", "OPTIONS"]`

**Frontend requirement:** All `fetch`/`axios` calls must use `credentials: "include"` (or `withCredentials: true`) to send the session cookie cross-origin.

---

# Endpoints

Source: `internal/pkg/server/delivery/routers/auth.go`

| # | Method | Path | Auth required | Handler |
|---|--------|------|:---:|---------|
| 1 | `POST` | `/api/auth/login/{provider}` | No | `Login` |
| 2 | `POST` | `/api/auth/logout` | Yes | `Logout` |
| 3 | `GET` | `/api/auth/check` | No | `CheckAuth` |
| 4 | `GET` | `/api/auth/identities` | Yes | `ListIdentities` |
| 5 | `POST` | `/api/auth/link/{provider}` | Yes | `LinkIdentity` |
| 6 | `DELETE` | `/api/auth/unlink/{provider}` | Yes | `UnlinkIdentity` |

Valid `{provider}` values: `vk`, `google`, `yandex`. Any other value returns `400` with `{"status": "Bad request"}`.

"Auth required" = routed through `LoginRequiredMiddleware`, which checks:
1. `session_id` cookie exists
2. Session is valid in Redis
3. `user.Status` is `""` or `"active"` (otherwise returns `403 USER_INACTIVE`)

---

## 1. POST /api/auth/login/{provider}

Authenticates the user via an OAuth provider and creates a session.

**Request:**

```
POST /api/auth/login/vk HTTP/1.1
Content-Type: application/json

{
  "code": "authorization_code_from_oauth",
  "state": "...",
  "codeVerifier": "...",
  "deviceID": "..."
}
```

| Field | Type | Required | Used by |
|-------|------|----------|---------|
| `code` | string | All providers | VK, Google, Yandex |
| `state` | string | VK only | VK (sent but **not verified** server-side) |
| `codeVerifier` | string | VK only | VK (PKCE) |
| `deviceID` | string | VK only | VK |

For **Google** and **Yandex**, only `code` is used. The other fields can be omitted or sent as empty strings.

**Success response (200):**

```json
{
  "isAuth": true,
  "user": {
    "id": 42,
    "displayName": "John Doe",
    "avatarUrl": "https://example.com/avatar.jpg",
    "status": "active"
  }
}
```

`Set-Cookie: session_id=<uuid>; Path=/; HttpOnly; SameSite=Lax; Expires=<now+720h>`
(plus `Secure` in production)

**Error responses:**

| HTTP | `status` field | Condition |
|------|---------------|-----------|
| 400 | `"Wrong JSON format"` | Malformed request body |
| 400 | `"Bad request"` | Unsupported `{provider}` value |
| 500 | `"VK server error"` | VK API returned an error |
| 500 | `"OAuth provider error"` | Google or Yandex returned an error |
| 500 | `"Server error"` | Any other internal error |

Source: `internal/pkg/auth/delivery/auth_handlers.go:35-83`

---

## 2. POST /api/auth/logout

Destroys the session and clears the cookie.

**Request:**

```
POST /api/auth/logout HTTP/1.1
Cookie: session_id=<uuid>
```

No body required.

**Success response (200):**

```json
{
  "isAuth": false,
  "user": {
    "id": 0,
    "displayName": ""
  }
}
```

`Set-Cookie: session_id=; Path=/; HttpOnly; SameSite=Lax; Expires=<yesterday>`

**Implementation recommendation:** The `user` field in the logout response contains a zero-value `User` struct (Go default: `id: 0`, `displayName: ""`). Frontend must **ignore the `user` object entirely** and rely solely on `isAuth: false`. On receiving this response, clear all local user state (store, context, cache) and redirect to the login screen.

**Error responses:**

| HTTP | `status` field | Condition |
|------|---------------|-----------|
| 401 | `"User not authorized"` | No/invalid session (from middleware) |
| 403 | `"USER_INACTIVE"` | User is banned/deleted (from middleware) |
| 500 | `"Server error"` | Redis error during session deletion |

Source: `internal/pkg/auth/delivery/auth_handlers.go:85-102`

---

## 3. GET /api/auth/check

Checks whether the current session is valid. Does **not** go through `LoginRequiredMiddleware` — it handles missing cookies gracefully.

**Request:**

```
GET /api/auth/check HTTP/1.1
Cookie: session_id=<uuid>
```

**Success response (200) — authenticated:**

```json
{
  "isAuth": true,
  "user": {
    "id": 42,
    "displayName": "John Doe",
    "avatarUrl": "https://example.com/avatar.jpg",
    "status": "active"
  }
}
```

**Success response (200) — not authenticated:**

```json
{
  "isAuth": false,
  "user": {
    "id": 0,
    "displayName": ""
  }
}
```

This endpoint always returns `200`. Check `isAuth` to determine auth state. No cookie is set or cleared.

Source: `internal/pkg/auth/delivery/auth_handlers.go:104-123`

---

## 4. GET /api/auth/identities

Returns all linked OAuth identities for the current user.

**Request:**

```
GET /api/auth/identities HTTP/1.1
Cookie: session_id=<uuid>
```

**Success response (200):**

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

| Field | Type | Always present | Notes |
|-------|------|:-:|-------|
| `id` | int | Yes | DB primary key |
| `userId` | int | Yes | Owner user ID |
| `provider` | string | Yes | `"vk"`, `"google"`, or `"yandex"` |
| `providerUserId` | string | Yes | Provider-specific user ID |
| `email` | string | No (`omitempty`) | May be empty if provider didn't return one |
| `createdAt` | string | No (`omitempty`) | ISO 8601 timestamp |
| `lastUsedAt` | string | No (`omitempty`) | ISO 8601 timestamp |

**Error responses:**

| HTTP | `status` field | Condition |
|------|---------------|-----------|
| 401 | `"User not authorized"` | No/invalid session |
| 403 | `"USER_INACTIVE"` | User banned/deleted |
| 500 | `"Server error"` | DB error |

Source: `internal/pkg/auth/delivery/auth_handlers.go:125-140`

---

## 5. POST /api/auth/link/{provider}

Links a new OAuth provider identity to the current user.

**Request:**

```
POST /api/auth/link/google HTTP/1.1
Content-Type: application/json
Cookie: session_id=<uuid>

{
  "code": "authorization_code_from_google"
}
```

Same `LoginRequest` body as `/login/{provider}`. For Google/Yandex only `code` is needed.

**Success response:** `204 No Content` (empty body)

**Error responses:**

| HTTP | `status` field | Condition |
|------|---------------|-----------|
| 400 | `"Wrong JSON format"` | Malformed body |
| 400 | `"Bad request"` | Unsupported provider |
| 400 | `"Identity already linked to another user"` | That OAuth account belongs to a different user |
| 401 | `"User not authorized"` | No/invalid session |
| 403 | `"USER_INACTIVE"` | User banned/deleted |
| 500 | `"Server error"` | OAuth or DB error |

Note: Linking an identity that is **already linked to the same user** is idempotent — the usecase calls `UpdateLastUsed` and returns success.

Source: `internal/pkg/auth/delivery/auth_handlers.go:142-177`

---

## 6. DELETE /api/auth/unlink/{provider}

Unlinks an OAuth provider identity from the current user.

**Request:**

```
DELETE /api/auth/unlink/vk HTTP/1.1
Cookie: session_id=<uuid>
```

No body required.

**Success response:** `204 No Content` (empty body)

**Error responses:**

| HTTP | `status` field | Condition |
|------|---------------|-----------|
| 400 | `"Cannot unlink last identity"` | Only one identity remains |
| 400 | `"Bad request"` | Identity not found for this provider |
| 401 | `"User not authorized"` | No/invalid session |
| 403 | `"USER_INACTIVE"` | User banned/deleted |
| 500 | `"Server error"` | DB error |

Source: `internal/pkg/auth/delivery/auth_handlers.go:179-204`

---

# Session & Cookies

Source: `internal/pkg/auth/delivery/session.go`

## Cookie attributes

| Attribute | Value | Notes |
|-----------|-------|-------|
| `Name` | `session_id` | |
| `Value` | UUID v4 string | Generated via `google/uuid` |
| `Path` | `/` | Available on all routes |
| `HttpOnly` | `true` | Not accessible via JS `document.cookie` |
| `Secure` | `true` in prod, `false` in dev | Controlled by `-prod` CLI flag |
| `SameSite` | `Lax` | Sent on top-level navigations and same-site requests |
| `Expires` | `now + cfg.Session.Duration` | Default: 720h (30 days) |

## Logout cookie

On logout, the same cookie is set with:
- `Value` = `""`
- `Expires` = yesterday (`time.Now().AddDate(0, 0, -1)`)

This causes the browser to delete the cookie.

## Session storage (Redis)

Sessions are stored as JSON in Redis with key = session UUID. The stored struct is `FullSessionData`:

```json
{
  "provider": "vk",
  "tokens": {
    "accessToken": "...",
    "refreshToken": "...",
    "idToken": "..."
  },
  "user": {
    "id": 42,
    "displayName": "John Doe",
    "avatarUrl": "...",
    "status": "active"
  }
}
```

The `provider` and `tokens` fields are **internal only** and are never sent to the client.

## SameSite=Lax and cross-domain caveat

The session cookie is set with `SameSite=Lax`. This works correctly when the frontend and API share the **same site** (same eTLD+1), for example:

- `app.example.com` (frontend) + `api.example.com` (backend) — **same site, works**
- `localhost:3000` (frontend) + `localhost:8080` (backend) — **same site, works**

If the frontend and API are on **different sites** (different eTLD+1, e.g. `myapp.com` + `api-myapp.io`), the browser will **not send** the `session_id` cookie on `fetch`/`XHR` requests, even with `credentials: "include"`. The cookie will only be sent on top-level navigations.

**NOTE:** If cross-site cookies are required, the backend would need to change to `SameSite=None; Secure` and the CORS config must include the exact frontend origin (no wildcard). This is a backend change outside the scope of this guide — coordinate with the backend team if your deployment topology requires it.

---

# Error model

All error responses use the same JSON envelope:

```json
{
  "status": "<error string>"
}
```

Source: `internal/models/responses.go` — `type ErrResponse struct { Status string json:"status" }`

## Error constants catalog

Source: `internal/pkg/server/delivery/responses/responses.go`

| Constant | Value | HTTP code | Context |
|----------|-------|-----------|---------|
| `ErrInternalServer` | `"Server error"` | 500 | Generic fallback |
| `ErrVKServer` | `"VK server error"` | 500 | VK API failure |
| `ErrOAuthProvider` | `"OAuth provider error"` | 500 | Google/Yandex failure |
| `ErrBadJSON` | `"Wrong JSON format"` | 400 | JSON decode error |
| `ErrBadRequest` | `"Bad request"` | 400 | Unsupported provider, identity not found |
| `ErrIdentityAlreadyLinked` | `"Identity already linked to another user"` | 400 | Link conflict |
| `ErrLastIdentity` | `"Cannot unlink last identity"` | 400 | Unlink blocked |
| `ErrNotAuthorized` | `"User not authorized"` | 401 | No/invalid session |
| `ErrUserInactive` | `"USER_INACTIVE"` | 403 | Banned or deleted user |

## Handling USER_INACTIVE (403)

Source: `internal/pkg/middleware/auth/login_required.go:30-33`

The middleware checks:
```
if user.Status != "" && user.Status != "active" {
    → 403 {"status": "USER_INACTIVE"}
}
```

This applies to **all protected endpoints** (logout, identities, link, unlink, and all non-auth protected routes).

**Frontend action:** Add a global response interceptor. On `403` with `status === "USER_INACTIVE"`, show an "account suspended" screen. This is distinct from `401` (redirect to login).

## Quick-reference: HTTP status → frontend action

| HTTP | Meaning | Frontend action |
|:----:|---------|-----------------|
| 200 | Success | Read `isAuth` / response body normally |
| 204 | Success (no body) | Link/unlink succeeded; refresh identity list if needed |
| 400 | Bad request | Show field-level or inline error from `status` field (bad JSON, unsupported provider, identity conflict, last identity) |
| 401 | Not authenticated | Session expired or missing. Clear local state, redirect to login |
| 403 | Forbidden (`USER_INACTIVE`) | Account banned or deleted. Show a dedicated "account suspended" screen. Do **not** redirect to login |
| 500 | Server error | Show generic error toast/banner. Optionally retry once. Log the `status` field for debugging |

---

# OAuth: VK flow

Source: `internal/pkg/auth/external/vk_api.go`

## Frontend steps

1. **Open VK authorization page** in the browser. Required parameters:
   - `client_id` — from VK app settings (same as backend `CLIENT_ID` env)
   - `redirect_uri` — must match backend `REDIRECT_URI` env
   - `code_challenge` — SHA256 of a random `code_verifier` (PKCE)
   - `code_challenge_method` — `S256`
   - `state` — random string for CSRF protection
   - `device_id` — persistent device identifier
   - `response_type` — `code`

2. **User authorizes** on VK, VK redirects back to `redirect_uri` with `?code=...&state=...&device_id=...`

3. **Frontend sends to backend:**
   ```json
   POST /api/auth/login/vk
   {
     "code": "<from VK redirect>",
     "state": "<from VK redirect>",
     "codeVerifier": "<original code_verifier, NOT the hash>",
     "deviceID": "<from VK redirect>"
   }
   ```

## Backend steps (what happens server-side)

1. **Exchange code for tokens** — `POST https://id.vk.com/oauth2/auth` with form params:
   - `grant_type=authorization_code`
   - `code`, `code_verifier`, `redirect_uri`, `client_id`, `device_id`, `state`
   - Returns: `access_token`, `refresh_token`, `id_token`

2. **Get public info** — `POST https://id.vk.com/oauth2/public_info` with form params:
   - `id_token`, `client_id`
   - Returns: user profile (first name, last name, avatar, email, user ID)

3. **Create/update user** in DB, create session in Redis, set cookie.

## VK PKCE details

VK is the **only** provider that uses PKCE. The flow:

1. **Frontend generates** a random `code_verifier` string (43-128 characters, `[A-Za-z0-9-._~]`).
2. **Frontend computes** `code_challenge = BASE64URL(SHA256(code_verifier))` (no padding).
3. **Frontend sends** `code_challenge` and `code_challenge_method=S256` to VK's authorization URL.
4. **After redirect**, frontend sends the **original `code_verifier`** (not the challenge) to the backend in the `codeVerifier` field.
5. **Backend forwards** `code_verifier` to VK's token endpoint. VK re-derives the challenge and compares.

The backend expects the **verifier**, not the challenge. Sending the challenge instead will cause VK to reject the token exchange.

### Frontend state check (recommended)

The backend does **not** verify the `state` parameter — it forwards it to VK unchanged (`vk_api.go:93`). CSRF protection must be implemented client-side:

1. Before redirecting to VK, generate a random `state` string and save it in `sessionStorage`.
2. After VK redirects back, compare `state` from the URL query with the saved value.
3. If they don't match, **do not call the backend** — show a CSRF error to the user.
4. If they match, proceed with `POST /api/auth/login/vk`.

## VK-specific notes

- `device_id` is a VK-specific parameter for device fingerprinting. VK returns it in the redirect URL; pass it through to the backend as `deviceID`.
- Error from VK → HTTP 500 with `{"status": "VK server error"}`.

---

# OAuth: Google flow

Source: `internal/pkg/auth/external/google_oauth.go`

## Frontend steps

1. **Open Google authorization page:**
   - `https://accounts.google.com/o/oauth2/v2/auth`
   - `client_id` — from Google Cloud Console (same as backend `GOOGLE_CLIENT_ID`)
   - `redirect_uri` — must match backend `GOOGLE_REDIRECT_URI`
   - `response_type=code`
   - `scope=openid email profile`
   - `state` — random CSRF token (managed client-side only)

2. **User authorizes**, Google redirects to `redirect_uri` with `?code=...`

3. **Frontend sends to backend:**
   ```json
   POST /api/auth/login/google
   {
     "code": "<from Google redirect>"
   }
   ```
   Only `code` is used. `state`, `codeVerifier`, `deviceID` are ignored.

## Backend steps

1. **Exchange code for tokens** — `POST https://oauth2.googleapis.com/token` with form params:
   - `code`, `client_id`, `client_secret`, `redirect_uri`, `grant_type=authorization_code`
   - `client_secret` is added server-side (not from the frontend request)

2. **Get user info** — `GET https://www.googleapis.com/oauth2/v3/userinfo` with `Authorization: Bearer <access_token>`
   - Returns: `sub`, `name`, `given_name`, `family_name`, `picture`, `email`

3. **Create/update user** in DB, create session in Redis, set cookie.

### Frontend state check (recommended)

The backend does **not** verify `state` for Google (the field is ignored entirely — `google_oauth.go:44` reads only `loginData.Code`). CSRF protection must be implemented client-side:

1. Before redirecting to Google, generate a random `state` string and save it in `sessionStorage`.
2. After Google redirects back, compare `state` from the URL query with the saved value.
3. If they don't match, **do not call the backend** — show a CSRF error to the user.
4. If they match, proceed with `POST /api/auth/login/google`.

## Google-specific notes

- No PKCE. The `client_secret` is stored server-side only — frontend never sees it.
- Error from Google → HTTP 500 with `{"status": "OAuth provider error"}`.

---

# OAuth: Yandex flow

Source: `internal/pkg/auth/external/yandex_oauth.go`

## Frontend steps

1. **Open Yandex authorization page:**
   - `https://oauth.yandex.ru/authorize`
   - `client_id` — from Yandex OAuth app (same as backend `YANDEX_CLIENT_ID`)
   - `response_type=code`
   - `state` — random CSRF token (managed client-side only)
   - `redirect_uri` — see VERIFY note below

2. **User authorizes**, Yandex redirects with `?code=...`

3. **Frontend sends to backend:**
   ```json
   POST /api/auth/login/yandex
   {
     "code": "<from Yandex redirect>"
   }
   ```
   Only `code` is used.

## Backend steps

1. **Exchange code for tokens** — `POST https://oauth.yandex.ru/token` with form params:
   - `grant_type=authorization_code`, `code`, `client_id`, `client_secret`
   - Note: **no `redirect_uri`** in the token exchange (unlike VK/Google)

2. **Get user info** — `GET https://login.yandex.ru/info?format=json` with `Authorization: OAuth <access_token>`
   - Note: Uses `OAuth ` prefix, not `Bearer ` (Yandex-specific)
   - Returns: `id`, `login`, `display_name`, `first_name`, `last_name`, `default_email`, `default_avatar_id`, `is_avatar_empty`

3. **Avatar URL construction:** If `is_avatar_empty` is `false` and `default_avatar_id` is present:
   `https://avatars.yandex.net/get-yapic/{default_avatar_id}/islands-200`
   Otherwise, `avatarUrl` will be empty.

4. **Create/update user** in DB, create session in Redis, set cookie.

### Frontend state check (recommended)

The backend does **not** verify `state` for Yandex (the field is ignored — `yandex_oauth.go:40` reads only `loginData.Code`). CSRF protection must be implemented client-side:

1. Before redirecting to Yandex, generate a random `state` string and save it in `sessionStorage`.
2. After Yandex redirects back, compare `state` from the URL query with the saved value.
3. If they don't match, **do not call the backend** — show a CSRF error to the user.
4. If they match, proceed with `POST /api/auth/login/yandex`.

### redirect_uri handling (VERIFY)

**VERIFY:** The backend does **not** send `redirect_uri` in the Yandex token exchange (`yandex_oauth.go:88-95` — only `grant_type`, `code`, `client_id`, `client_secret`). There is also **no `YANDEX_REDIRECT_URI` env var** in `config.go`. This means:

- The redirect URI is configured **only** in the Yandex OAuth app console (allowlisted callback URL).
- The frontend must redirect to **exactly** the URL registered in the Yandex console.
- To verify your setup:
  1. Open the [Yandex OAuth app settings](https://oauth.yandex.ru/) and check the "Callback URI" field.
  2. Ensure the frontend authorization URL uses that same `redirect_uri`.
  3. **NOTE:** Since the backend omits `redirect_uri` from the token exchange, Yandex will accept the code as long as it was issued to the registered callback.

## Yandex-specific notes

- No PKCE. `client_secret` stored server-side — frontend never sees it.
- Error from Yandex → HTTP 500 with `{"status": "OAuth provider error"}`.

---

# Identity linking / unlinking

## Linking flow

1. User is already logged in (has a valid `session_id` cookie).
2. Frontend initiates OAuth with the new provider (same flow as login: open auth page, get code).
3. Frontend calls `POST /api/auth/link/{provider}` with the `LoginRequest` body.
4. Backend authenticates with the provider, then creates a `user_identity` row linking the provider to the current user.
5. On success: `204 No Content`.

**Edge cases:**
- If the identity is already linked to the **same user** → success (idempotent, updates `last_used_at`).
- If the identity is already linked to a **different user** → `400 "Identity already linked to another user"`.

## Unlinking flow

1. Frontend calls `DELETE /api/auth/unlink/{provider}`.
2. Backend checks the user has more than one identity.
3. On success: `204 No Content`.

**Edge cases:**
- Last remaining identity → `400 "Cannot unlink last identity"`. Frontend should disable the unlink button when only 1 identity is listed.
- Identity not found → `400 "Bad request"`.

---

# Frontend implementation checklist

- [ ] Configure `fetch`/`axios` with `credentials: "include"` for all API calls
- [ ] Implement VK OAuth with PKCE (`code_verifier` / `code_challenge`)
- [ ] Implement Google OAuth (standard authorization code)
- [ ] Implement Yandex OAuth (standard authorization code)
- [ ] Call `POST /api/auth/login/{provider}` with the correct body per provider
- [ ] Read `isAuth` and `user` from login/check responses
- [ ] Handle `user.displayName` (was `name`) and `user.avatarUrl` (was `avatar`, nullable)
- [ ] Handle `user.status` field — treat empty/missing as `"active"`
- [ ] Add global 401 handler → redirect to login
- [ ] Add global 403 handler → check for `"USER_INACTIVE"` → show suspended screen
- [ ] Implement `GET /api/auth/check` on app load to restore session state
- [ ] Implement `POST /api/auth/logout`
- [ ] Implement identity management UI: list (`GET /api/auth/identities`), link, unlink
- [ ] Disable unlink button when only 1 identity exists
- [ ] Handle `"Identity already linked to another user"` error in link flow

---

# Environment / config checklist

Environment variables required for auth (from `internal/pkg/config/config.go`):

### VK

| Env var | Config field | Required |
|---------|-------------|:---:|
| `REDIRECT_URI` | `VKApiConfig.RedirectURI` | Yes |
| `CLIENT_ID` | `VKApiConfig.ClientID` | Yes |
| `SECRET_KEY` | `VKApiConfig.SecretKey` | Yes |
| `SERVICE_KEY` | `VKApiConfig.ServiceKey` | Yes |

VK endpoint URLs are in `config.yaml` under `vk_api.exchange.url` and `vk_api.public_info.url`.

### Google

| Env var | Config field | Required |
|---------|-------------|:---:|
| `GOOGLE_CLIENT_ID` | `GoogleOAuthConfig.ClientID` | Yes |
| `GOOGLE_CLIENT_SECRET` | `GoogleOAuthConfig.ClientSecret` | Yes |
| `GOOGLE_REDIRECT_URI` | `GoogleOAuthConfig.RedirectURI` | Yes |

Google endpoint URLs are hardcoded constants in `google_oauth.go`.

### Yandex

| Env var | Config field | Required |
|---------|-------------|:---:|
| `YANDEX_CLIENT_ID` | `YandexOAuthConfig.ClientID` | Yes |
| `YANDEX_CLIENT_SECRET` | `YandexOAuthConfig.ClientSecret` | Yes |

Yandex endpoint URLs are hardcoded constants in `yandex_oauth.go`. Note: Yandex has **no `redirect_uri` env var** — it is configured in the Yandex OAuth app settings directly.

### Session

| Env var / YAML | Config field | Default |
|----------------|-------------|---------|
| `SESSION_DURATION` (env) or `session.duration` (YAML) | `SessionConfig.Duration` | `720h` |

### Redis (session storage)

| Env var | Config field |
|---------|-------------|
| `REDIS_PASSWORD` | `RedisConfig.Password` |
| `REDIS_HOST` | `RedisConfig.Host` |
| `REDIS_PORT` | `RedisConfig.Port` |
| `REDIS_DB` | `RedisConfig.DB` |

### Server / CORS

Configured in `config.yaml`:

```yaml
server:
  port: 8080
  origins:
    - http://localhost:3000    # Add your frontend origin here
  headers:
    - X-Requested-With
    - Content-Type
  methods:
    - GET
    - POST
    - DELETE
    - PUT
    - HEAD
    - OPTIONS
```

---

# Smoke tests (curl)

## Login (VK)

```bash
curl -v -X POST http://localhost:8080/api/auth/login/vk \
  -H "Content-Type: application/json" \
  -d '{"code":"TEST_CODE","state":"TEST_STATE","codeVerifier":"TEST_VERIFIER","deviceID":"TEST_DEVICE"}' \
  -c cookies.txt
```

Expected: `200` with `{"isAuth":true,"user":{...}}` and a `Set-Cookie: session_id=...` header.
(Will return `500` with a real VK error unless you have a valid authorization code.)

## Login (Google)

```bash
curl -v -X POST http://localhost:8080/api/auth/login/google \
  -H "Content-Type: application/json" \
  -d '{"code":"TEST_CODE"}' \
  -c cookies.txt
```

## Login (Yandex)

```bash
curl -v -X POST http://localhost:8080/api/auth/login/yandex \
  -H "Content-Type: application/json" \
  -d '{"code":"TEST_CODE"}' \
  -c cookies.txt
```

## Check auth

```bash
curl -v http://localhost:8080/api/auth/check \
  -b cookies.txt
```

Expected: `200` with `{"isAuth":true,"user":{...}}` if session is valid, or `{"isAuth":false,"user":{"id":0,"displayName":""}}` if not.

## List identities

```bash
curl -v http://localhost:8080/api/auth/identities \
  -b cookies.txt
```

Expected: `200` with `[{"id":1,"userId":42,"provider":"vk",...}, ...]`

## Link identity

```bash
curl -v -X POST http://localhost:8080/api/auth/link/google \
  -H "Content-Type: application/json" \
  -d '{"code":"GOOGLE_AUTH_CODE"}' \
  -b cookies.txt
```

Expected: `204 No Content`

## Unlink identity

```bash
curl -v -X DELETE http://localhost:8080/api/auth/unlink/vk \
  -b cookies.txt
```

Expected: `204 No Content` (or `400` if last identity)

## Logout

```bash
curl -v -X POST http://localhost:8080/api/auth/logout \
  -b cookies.txt -c cookies.txt
```

Expected: `200` with `{"isAuth":false,"user":{"id":0,"displayName":""}}` and cookie cleared.

## Unsupported provider

```bash
curl -v -X POST http://localhost:8080/api/auth/login/facebook \
  -H "Content-Type: application/json" \
  -d '{"code":"test"}'
```

Expected: `400` with `{"status":"Bad request"}`

## Bad JSON

```bash
curl -v -X POST http://localhost:8080/api/auth/login/vk \
  -H "Content-Type: application/json" \
  -d '{invalid'
```

Expected: `400` with `{"status":"Wrong JSON format"}`

---

# Doc mismatches found

Discrepancies between existing documentation and the actual codebase:

| # | Document | Claim | Actual (from code) | Severity |
|---|----------|-------|--------------------|----------|
| 1 | `Frontend API field changes backlog.md` (old version, pre-audit) | Listed `provider` as a new field on `AuthResponse` | `AuthResponse` has only `isAuth` and `user`. `provider` exists only in `FullSessionData` (Redis internal, never sent to client). | Fixed in backlog audit |
| 2 | `Frontend API field changes backlog.md` (old version, pre-audit) | Implied `provider` is a request body field | `provider` is a URL path variable `{provider}` extracted via `mux.Vars(r)["provider"]`. `LoginRequest` has only `code`, `state`, `codeVerifier`, `deviceID`. | Fixed in backlog audit |
| 3 | None | VK `state` parameter is sent to VK API but never verified by backend | `vk_api.go:93` sets `state` in the token exchange request but there is no comparison against a stored value. CSRF protection relies on frontend-only `state` verification. | Informational |
| 4 | None | Google and Yandex ignore `state`, `codeVerifier`, `deviceID` from `LoginRequest` | `google_oauth.go:44` uses only `loginData.Code`; `yandex_oauth.go:40` uses only `loginData.Code`. Other fields are silently ignored. | Informational |
