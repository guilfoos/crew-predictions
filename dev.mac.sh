#!/usr/bin/env bash
set -e

# Kill whatever is on emulator + server ports (including orphaned Java processes)
lsof -t -i tcp:8080,8091,9099,4000,4400,4500 | xargs kill -9 2>/dev/null || true
sleep 1

# Start Firestore + Auth emulators in background
firebase emulators:start --only firestore,auth &
EMULATOR_PID=$!

# Wait for emulators to be ready
sleep 5

# Start Go server
FIRESTORE_EMULATOR_HOST=127.0.0.1:8091 \
FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099 \
GOOGLE_CLOUD_PROJECT=crew-predictions \
/opt/homebrew/bin/go run ./cmd/server

# Clean up emulators (and their Java child processes) on exit
trap "lsof -t -i tcp:8091,9099 | xargs kill -9 2>/dev/null || true; kill $EMULATOR_PID 2>/dev/null || true" EXIT
