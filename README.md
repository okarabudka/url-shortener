# url-shortener

[![CI](https://github.com/okarabudka/url-shortener/actions/workflows/ci.yml/badge.svg)](https://github.com/okarabudka/url-shortener/actions/workflows/ci.yml)

A small URL shortener REST service in Go: create a short link, follow it, delete it. SQLite storage, structured
logging, request validation, Basic Auth on write operations, a Docker image and CI.

## Why it exists

It started in May 2024 as a Go learning project by a PHP backend developer and was left unfinished. It has since been
completed as an example of a small Go service built the way a production one would be: the standard
`cmd`/`internal`/`configs` layout, configuration from YAML with environment overrides, narrow interfaces between the
layers, correct HTTP status codes and timeouts, graceful shutdown, and tests that run without network or real waiting.
The layout follows [go-architecture](https://github.com/okarabudka/go-architecture).

## What you get

**Three endpoints.**

| Method   | Path            | Auth  | Success                               | Errors                                                      |
|----------|-----------------|-------|---------------------------------------|-------------------------------------------------------------|
| `POST`   | `/url`          | Basic | 201 `{"status":"OK","alias":"..."}`   | 400 invalid request, 401 no auth, 409 alias taken           |
| `GET`    | `/{alias}`      | none  | 302 with `Location: <url>`            | 404 unknown alias                                           |
| `DELETE` | `/url/{alias}`  | Basic | 204                                   | 401 no auth, 404 unknown alias                              |

`POST /url` takes `{"url": "https://...", "alias": "optional"}`. Only `http` and `https` URLs are accepted. An alias
has at most 32 letters and digits; without one, a random alias of `alias_length` characters is generated with
`crypto/rand`. Errors come back as `{"status":"Error","error":"..."}`.

**Closed by default.** Creating and deleting links needs Basic Auth. The credentials come from `HTTP_USER` and
`HTTP_PASSWORD` only, never from a config file, and the service refuses to start without them.

**Production basics.** Server timeouts from the config, graceful shutdown on `SIGINT`/`SIGTERM`, one structured log
line per request with a request id, a multi-stage Docker image that runs as an unprivileged user, and data on a named
volume that survives restarts.

**Quality checks.** `go vet`, golangci-lint (with gosec, revive, bodyclose, gofmt and goimports) and `go test -race`
run in GitHub Actions together with a secret scan (gitleaks) and a smoke test of the Docker Compose setup. Dependabot
keeps Go modules, actions and base images current.

## How to use it

Requirements: Docker and make. Go is only needed to run the service without Docker.

### Run with Docker Compose

```bash
make up            # copies .env.example to .env, builds and starts the service
make smoke         # creates, follows and deletes a link against it
make down
```

Change `HTTP_USER` and `HTTP_PASSWORD` in `.env` before exposing the service. The service listens on
`localhost:8080`; set `HTTP_PORT` in `.env` to use another host port. Run `make help` for all targets.

Create a link, follow it, delete it (the credentials come from `.env`):

```bash
set -a; . ./.env; set +a

curl -i -u "$HTTP_USER:$HTTP_PASSWORD" -X POST http://localhost:8080/url \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com", "alias": "example"}'

curl -i http://localhost:8080/example

curl -i -u "$HTTP_USER:$HTTP_PASSWORD" -X DELETE http://localhost:8080/url/example
```

A session against the running container (headers `Date` and `Content-Length` trimmed):

```text
$ curl -i -u "$HTTP_USER:$HTTP_PASSWORD" -X POST http://localhost:8080/url -H 'Content-Type: application/json' -d '{"url": "https://example.com", "alias": "example"}'
HTTP/1.1 201 Created
Content-Type: application/json

{"status":"OK","alias":"example"}

$ curl -i http://localhost:8080/example
HTTP/1.1 302 Found
Content-Type: text/html; charset=utf-8
Location: https://example.com

<a href="https://example.com">Found</a>.

$ curl -i -u "$HTTP_USER:$HTTP_PASSWORD" -X DELETE http://localhost:8080/url/example
HTTP/1.1 204 No Content
```

### Run without Docker

Requirements: Go 1.27 and a C compiler (go-sqlite3 uses CGO).

```bash
mkdir -p storage
HTTP_USER=admin HTTP_PASSWORD=change-me go run ./cmd/app
```

### Configuration

The service reads `configs/config.yaml`, or the file in `CONFIG_PATH`. The Docker image uses
`configs/config.docker.yaml`.

| Key                 | Environment     | Default                | Meaning                                       |
|---------------------|-----------------|------------------------|-----------------------------------------------|
| `env`               |                 | `local`                | `local`, `dev` (debug logs) or `prod`         |
| `db_storage_path`   | `STORAGE_PATH`  | required               | SQLite database file                          |
| `alias_length`      |                 | `6`                    | length of generated aliases                   |
| `http.address`      | `HTTP_ADDRESS`  | `localhost:8080`       | listen address                                |
| `http.timeout`      |                 | `4s`                   | read, read-header and write timeout           |
| `http.idle_timeout` |                 | `60s`                  | keep-alive idle timeout                       |
|                     | `HTTP_USER`     | required               | Basic Auth user for `POST` and `DELETE`       |
|                     | `HTTP_PASSWORD` | required               | Basic Auth password for `POST` and `DELETE`   |

Environment variables override the file.

### Testing

```bash
make test          # go test -race -cover ./... in a golang container
make lint          # go vet and golangci-lint in containers
```

Handlers are tested with `httptest` and hand-written fakes of their storage interfaces, the storage against a
temporary SQLite file, and `internal/app` end to end: create, follow, delete, follow again, plus the 401 cases.

## Repository layout

```
cmd/app/main.go                  entry point: config, logger, app.Run with signal handling
configs/                         config.yaml for local runs, config.docker.yaml for the image
internal/app/                    wiring: storage, router, http.Server, graceful shutdown
internal/config/                 config struct, loading and validation
internal/http/handlers/          save (POST /url), redirect (GET /{alias}), remove (DELETE /url/{alias})
internal/http/middleware/logger/ request log line
internal/lib/                    JSON responses, slog helper, random alias generator
internal/storage/                storage errors and the SQLite implementation
scripts/smoke.sh                 curl smoke test used by make smoke and CI
```

## Status

Maintained.

## License

MIT, see [LICENSE](LICENSE).
