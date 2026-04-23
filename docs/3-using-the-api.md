---
description: Overview of the JSONAir API.
---

# 3. Using the API

The JSONAir API is a simple, read-only HTTP API. Agents use it to retrieve configuration data. There are no write endpoints — JSONAir is intentionally uni-directional.

---

## Base URL

All endpoints are versioned under:

```
/api/v1/jsonair/
```

---

## Authentication Flow

The API uses a two-step authentication model:

1. **Exchange your PAT for a short-lived JWT.** Send your plain-text Personal Access Token (PAT) to the `/auth/token` endpoint. The server hashes it, looks it up in the database, and returns a signed JWT.

2. **Use the JWT as a Bearer token.** Include the JWT in the `Authorization` header on all subsequent requests. When it expires, repeat step one.

This means your long-lived PAT is only ever sent once per session, and the short-lived JWT is what travels with every API call.

---

## Endpoints at a Glance

| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| `POST` | `/api/v1/jsonair/auth/token` | No | Exchange a PAT for a JWT |
| `GET` | `/api/v1/jsonair/config` | Yes | Retrieve configuration data |
| `GET` | `/api/v1/jsonair/reload` | Yes | Retrieve the reload key |
| `GET` | `/api/v1/jsonair/debug` | Yes | Retrieve the debug level |

---

## Content Type

All request bodies must be JSON. All responses are JSON or plain text depending on the endpoint.

```
Content-Type: application/json
```

---

## Input Validation

The `type` and `name` fields in all requests are sanitized server-side. Only the following characters are accepted — any others are silently stripped:

```
a-z  A-Z  0-9  -  _  .
```
