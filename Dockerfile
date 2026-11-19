# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS builder

# mattn/go-sqlite3 needs CGO and a C toolchain.
RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/url-shortener ./cmd/app

FROM alpine:3.24

RUN addgroup -S app && adduser -S -G app app \
    && mkdir /data && chown app:app /data

COPY --from=builder /out/url-shortener /usr/local/bin/url-shortener
COPY configs/config.docker.yaml /etc/url-shortener/config.yaml

ENV CONFIG_PATH=/etc/url-shortener/config.yaml
USER app
VOLUME /data
EXPOSE 8080

ENTRYPOINT ["url-shortener"]
