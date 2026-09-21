---
description: An overview of installing JSONAir and how its programs fit together.
---

# 2. Installation

JSONAir is a small set of standalone programs that share one database. You only need to run the ones you use.

| Program | Required? | What it does | Default port |
|---------|-----------|--------------|--------------|
| `jsonair` | Yes | The read-only API. Serves configurations to agents and applications. | `9191` |
| `jsonair-agent` | No | Runs next to your application. Keeps a local config file in sync with `jsonair` and runs a reload command when it changes. | — |
| `jsonair-admin` | No | A web interface for people to create, edit and delete configurations. | `8080` |
| `jsonair-write` | No | A write-only API for other services to create, update and delete configurations. Meant for an internal network. | `9192` |
| `jsonair-encrypt` | No | A command-line tool to encrypt configuration data for manual database inserts. | — |

---

## Suggested Order

1. [2.2 Dependencies](2.2-dependancies.md) — what you need installed.
2. [2.3 Compiling JSONAir](2.3-compiling-jsonair.md) — build the programs.
3. [2.4 Database Setup](2.4-database-setup.md) — create the database, import the schema, add a key.
4. [2.5 Configuring and Running JSONAir](2.5-configuration.md) — start the read API.
5. [2.6 Configuring and Running the Agent](2.6-agent-configuration.md) — keep a local file in sync.
6. Optionally, a way to manage configurations:
   * [2.7 Encrypting Configuration Data](2.7-encrypting-config-data.md) — by hand, with `jsonair-encrypt`.
   * [2.8 Admin Web Interface](2.8-admin-web-interface.md) — for people.
   * [2.9 The Write API](2.9-write-api.md) — for other services.

Once it is running, see [3. Using the API](3-using-the-api.md).

---

## Which Secrets Go Where

Each program has its own secrets. Generate every one with `openssl rand -hex 32`, and never reuse a value between two names below, with one exception: `CONFIG_ENCRYPT_SECRET`.

| Secret | `jsonair` | `jsonair-admin` | `jsonair-write` | `jsonair-encrypt` |
|--------|:---------:|:---------------:|:---------------:|:-----------------:|
| `CONFIG_ENCRYPT_SECRET` | ✓ | ✓ | ✓ | ✓ |
| `JWT_TOKEN_SECRET` | ✓ | | | |
| `TOKEN_HMAC_SECRET` | ✓ | | | |
| `ADMIN_SESSION_SECRET` | | ✓ | | |
| `WRITE_JWT_TOKEN_SECRET` | | | ✓ | |
| `WRITE_TOKEN_HMAC_SECRET` | | | ✓ | |

`CONFIG_ENCRYPT_SECRET` is the one value that **must be identical** everywhere it appears. It encrypts configuration data at rest, so whichever program writes a configuration and whichever reads it must agree. See [2.7](2.7-encrypting-config-data.md).

---

## Network Exposure

* `jsonair` (the read API) is the only program intended to be reachable by many clients, and may listen on a public interface. Use TLS.
* `jsonair-admin` and `jsonair-write` can change your configurations. Keep both on an internal network or behind a VPN. For `jsonair-write`, also consider requiring client certificates (`HTTP_CLIENT_CA`).
