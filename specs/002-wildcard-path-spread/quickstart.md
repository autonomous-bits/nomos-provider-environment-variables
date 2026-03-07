# Quickstart: Wildcard Path Spread Operator

**Feature**: `002-wildcard-path-spread`  
**Date**: 2026-03-07

---

## Prerequisites

- Provider initialised with feature 001 configuration.
- Go 1.26+, `grpc` client connected to the running provider.

---

## Scenario 1 — Retrieve all variables (root wildcard)

**Setup:**
```bash
export API_KEY=secret123
export DB_HOST=localhost
export DB_PORT=5432
```

**Provider init config:**
```json
{
  "separator": "_",
  "case_transform": "upper"
}
```

**gRPC request:**
```
FetchRequest { path: ["*"] }
```

**Response:**
```json
{
  "API_KEY": "secret123",
  "DB_HOST":  "localhost",
  "DB_PORT":  5432
}
```

> Note: `DB_PORT` is `5432` (number) when `enable_type_conversion: true`.

---

## Scenario 2 — Retrieve all variables under a namespace

**Setup:**
```bash
export DATABASE_HOST=localhost
export DATABASE_PORT=5432
export DATABASE_URL=postgres://localhost/mydb
export APP_VERSION=2.0
```

**Provider init config:**
```json
{
  "separator": "_",
  "case_transform": "upper"
}
```

**gRPC request:**
```
FetchRequest { path: ["database", "*"] }
```

**Response:**
```json
{
  "HOST": "localhost",
  "PORT": 5432,
  "URL":  "postgres://localhost/mydb"
}
```

> `APP_VERSION` is excluded — it does not start with `DATABASE_`.

---

## Scenario 3 — Deeply nested namespace wildcard

**Setup:**
```bash
export APP_DATABASE_HOST=localhost
export APP_DATABASE_PORT=5432
export APP_CACHE_HOST=redis
```

**gRPC request:**
```
FetchRequest { path: ["app", "database", "*"] }
```

**Response:**
```json
{
  "HOST": "localhost",
  "PORT": 5432
}
```

> `APP_CACHE_HOST` is excluded — it does not start with `APP_DATABASE_`.

---

## Scenario 4 — Root wildcard with provider prefix

**Setup:**
```bash
export MYAPP_API_KEY=secret
export MYAPP_DB_HOST=localhost
export SYSTEM_PATH=/usr/bin
```

**Provider init config:**
```json
{
  "separator": "_",
  "case_transform": "upper",
  "prefix": "MYAPP_",
  "prefix_mode": "prepend"
}
```

**gRPC request:**
```
FetchRequest { path: ["*"] }
```

**Response:**
```json
{
  "API_KEY": "secret",
  "DB_HOST": "localhost"
}
```

> `SYSTEM_PATH` is excluded — it does not start with `MYAPP_`. The `MYAPP_` prefix is stripped from all returned keys.

---

## Scenario 5 — Empty namespace (no matching variables)

**Setup:** No variables with prefix `CACHE_` are set.

**gRPC request:**
```
FetchRequest { path: ["cache", "*"] }
```

**Response:**
```json
{}
```

> An empty struct is returned. This is not an error.

---

## Scenario 6 — Invalid wildcard placement (error)

**gRPC request:**
```
FetchRequest { path: ["*", "host"] }
```

**Response:**
```
gRPC status: INVALID_ARGUMENT
message: "wildcard operator '*' is only valid at the terminal position of a path; found at index 0"
```
