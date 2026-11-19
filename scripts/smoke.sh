#!/bin/sh
# Smoke test against a running service: create, follow, delete and follow a link again.
# Usage: scripts/smoke.sh [base-url]   (credentials from HTTP_USER and HTTP_PASSWORD)
set -eu

BASE=${1:-http://localhost:${HTTP_PORT:-8080}}
AUTH="${HTTP_USER:?HTTP_USER is not set}:${HTTP_PASSWORD:?HTTP_PASSWORD is not set}"
ALIAS="smoke$(date +%s)"

# check <expected status> <description> <curl args...>
check() {
    expected=$1 description=$2
    shift 2
    actual=$(curl -s -o /dev/null -w '%{http_code}' "$@")
    if [ "$actual" != "$expected" ]; then
        echo "FAIL $description: expected $expected, got $actual" >&2
        exit 1
    fi
    echo "ok   $description: $actual"
}

wait_for_service() {
    for _ in $(seq 1 30); do
        curl -s -o /dev/null "$BASE/" && return 0
        sleep 1
    done
    echo "FAIL service at $BASE did not answer" >&2
    exit 1
}

wait_for_service

check 401 "create without auth" -X POST "$BASE/url" -d "{\"url\":\"https://example.com\",\"alias\":\"$ALIAS\"}"
check 400 "create with invalid url" -u "$AUTH" -X POST "$BASE/url" -d '{"url":"javascript:alert(1)"}'
check 201 "create" -u "$AUTH" -X POST "$BASE/url" -d "{\"url\":\"https://example.com\",\"alias\":\"$ALIAS\"}"
check 409 "create with taken alias" -u "$AUTH" -X POST "$BASE/url" -d "{\"url\":\"https://example.com\",\"alias\":\"$ALIAS\"}"
check 302 "follow" "$BASE/$ALIAS"

location=$(curl -s -o /dev/null -D - "$BASE/$ALIAS" | tr -d '\r' | sed -n 's/^[Ll]ocation: //p')
[ "$location" = "https://example.com" ] || { echo "FAIL location: $location" >&2; exit 1; }
echo "ok   location: $location"

check 204 "delete" -u "$AUTH" -X DELETE "$BASE/url/$ALIAS"
check 404 "delete again" -u "$AUTH" -X DELETE "$BASE/url/$ALIAS"
check 404 "follow after delete" "$BASE/$ALIAS"
