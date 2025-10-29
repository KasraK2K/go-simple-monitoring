#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CGO_FLAGS="-Wno-gnu-folding-constant"

if ! command -v go >/dev/null 2>&1; then
    echo "[dev] go toolchain is required" >&2
    exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
    echo "[dev] python3 is required" >&2
    exit 1
fi

if [[ -n "${DEV_CMD:-}" ]]; then
    # shellcheck disable=SC2206
    CMD=(${DEV_CMD})
else
    CMD=(go run ./cmd)
fi

cleanup() {
    if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
        echo "[dev] stopping server (pid $SERVER_PID)"
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    unset SERVER_PID || true
}

start_server() {
    echo "[dev] starting server"
    (
        cd "$ROOT_DIR"
        export CGO_CFLAGS="$CGO_FLAGS"
        "${CMD[@]}"
    ) &
    SERVER_PID=$!
}

restart_server() {
    cleanup
    start_server
}

calc_hash() {
    ROOT_DIR="$ROOT_DIR" python3 <<'PY'
import hashlib
import os
import sys

root = os.environ["ROOT_DIR"]
extensions = {'.go', '.html', '.json'}
ignore_dirs = {'.git', 'node_modules', 'vendor', '__pycache__', 'dist', 'build'}
sha1 = hashlib.sha1()

for current_root, dirs, files in os.walk(root):
    dirs[:] = [d for d in dirs if d not in ignore_dirs and not d.startswith('.')]
    matched = []
    for filename in files:
        _, ext = os.path.splitext(filename)
        if ext.lower() in extensions:
            matched.append(filename)
    for filename in sorted(matched):
        path = os.path.join(current_root, filename)
        rel = os.path.relpath(path, root)
        sha1.update(rel.encode('utf-8', errors='ignore'))
        try:
            with open(path, 'rb') as f:
                for chunk in iter(lambda: f.read(8192), b''):
                    sha1.update(chunk)
        except OSError:
            continue

print(sha1.hexdigest())
PY
}

trap cleanup EXIT INT TERM

last_hash=$(calc_hash)
start_server

while true; do
    sleep 1
    new_hash=$(calc_hash)
    if [[ "$new_hash" != "$last_hash" ]]; then
        echo "[dev] change detected; restarting server"
        last_hash="$new_hash"
        restart_server
    elif [[ -n "${SERVER_PID:-}" ]] && ! kill -0 "$SERVER_PID" 2>/dev/null; then
        echo "[dev] server exited; waiting for file changes to restart"
        unset SERVER_PID || true
        sleep 1
    fi
done
