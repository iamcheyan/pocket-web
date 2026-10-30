#!/bin/sh
set -eu
cd "$(dirname "$0")"
(cd backend && go run .) &
backend_pid=$!
(cd frontend && npm run dev -- --host 127.0.0.1) &
frontend_pid=$!
stop() {
  kill "$backend_pid" "$frontend_pid" 2>/dev/null || true
  wait "$backend_pid" "$frontend_pid" 2>/dev/null || true
}
trap stop INT TERM EXIT
wait -n "$backend_pid" "$frontend_pid"
