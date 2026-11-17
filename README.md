# go-short-urls

A URL shortener REST service in Go with SQLite storage, structured logging and request validation.

## Context

Built in May 2024 as a Go learning project by a PHP backend developer. It explores how a typical small backend service is put together in Go: configuration loading, structured logging with `log/slog`, a router with middleware, a storage layer behind an interface, and JSON request validation. The project was developed over a few days and is unfinished.

## Features

- `POST /url` saves a URL under an alias and returns the alias as JSON.
  - If no alias is given, a random 6-character alphanumeric alias is generated.
  - The URL is validated (`required`, `url`); validation errors are returned in a readable form.
  - Duplicate aliases are rejected with `url already exists`.
- SQLite storage with a unique, indexed `alias` column; the table is created on startup.
- Storage layer also provides `GetURL(alias)` (not yet exposed over HTTP).
- Environment-dependent logging (`local`, `dev`: debug level; `prod`: info level).
- Request logging middleware: method, path, request ID, status, bytes written and duration per request.
- YAML configuration with defaults, path overridable via `CONFIG_PATH`.

Example request and responses:

```json
POST /url
{"url": "https://example.com/some/long/path", "alias": "example"}

{"status": "OK", "alias": "example"}
{"status": "Error", "error": "url already exists"}
```

## Tech stack

- Go 1.22
- [chi v5](https://github.com/go-chi/chi) router and middleware, [go-chi/render](https://github.com/go-chi/render) for JSON
- [go-playground/validator](https://github.com/go-playground/validator) for request validation
- [cleanenv](https://github.com/ilyakaznacheev/cleanenv) for YAML/env configuration
- SQLite via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) (requires CGO)
- `log/slog` for structured logging
- [testify](https://github.com/stretchr/testify) for assertions

## Architecture

```
cmd/app/main.go                  entry point: config, logger, storage, router, HTTP server
config/config.yaml               default configuration
internal/config/                 config struct and loader (MustLoad)
internal/handlers/save/          POST /url handler
internal/storage/                storage errors (ErrURLExists, ErrURLNotFound)
internal/storage/sqlite/         SQLite implementation
internal/middleware/logger/      request logging middleware
internal/lib/api/response/       common JSON response and validation error formatting
internal/lib/random/             random alias generator
```

Key decisions:

- Handlers depend on small interfaces (for example `URLSaver`) rather than the concrete storage, so the storage can be swapped or mocked. A `go:generate` directive for mockery is prepared in the save handler.
- Storage errors are wrapped with the operation name (`storage.sqlite.SaveURL: ...`) and mapped to domain errors, so handlers do not depend on SQLite error codes.
- All application code lives under `internal/`.

## Getting started

Requirements: Go 1.22 and a C compiler (go-sqlite3 uses CGO).

```bash
mkdir -p storage          # database file: ./storage/storage.db (see config/config.yaml)
go run ./cmd/app
```

The server listens on `localhost:8080` by default. A different config file can be used with:

```bash
CONFIG_PATH=/path/to/config.yaml go run ./cmd/app
```

Save a URL:

```bash
curl -X POST http://localhost:8080/url \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com", "alias": "example"}'
```

## Testing

A table-driven test for the alias generator exists in `internal/lib/random/random.test.go`. Its file name does not end in `_test.go`, so `go test ./...` does not pick it up yet; renaming it to `random_test.go` would enable it. There are no other tests.

## Status

Unfinished learning project, developed in May 2024 and not actively maintained. Open points visible in the code:

- No redirect endpoint yet (`GetURL` exists only in the storage layer); `DeleteURL` is a TODO.
- The HTTP server timeouts in `cmd/app/main.go` are set as plain integers (`4`, `30`), which Go interprets as nanoseconds, not seconds.

## License

MIT, see [LICENSE](LICENSE).
