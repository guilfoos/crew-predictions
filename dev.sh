#!/usr/bin/env bash
set -e

# Kill whatever is on emulator + server ports (including orphaned Java processes)
fuser -k 8080/tcp 8091/tcp 9099/tcp 4000/tcp 4400/tcp 4500/tcp 2>/dev/null || true
sleep 1

# Start Firestore + Auth emulators in background
firebase emulators:start --only firestore,auth &
EMULATOR_PID=$!

# Wait for emulators to be ready
sleep 5

# Start Go server
FIRESTORE_EMULATOR_HOST=localhost:8091 \
FIREBASE_AUTH_EMULATOR_HOST=localhost:9099 \
GOOGLE_CLOUD_PROJECT=crew-predictions \
/usr/local/go/bin/go run ./cmd/server

# Clean up emulators (and their Java child processes) on exit
trap "fuser -k 8091/tcp 9099/tcp 2>/dev/null || true; kill $EMULATOR_PID 2>/dev/null || true" EXIT
