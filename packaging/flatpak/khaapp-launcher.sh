#!/bin/bash
set -e

DAEMON_BIN="${FLATPAK_DEST:-/app}/bin/khaapp-daemon"
FRONTEND_BIN="${FLATPAK_DEST:-/app}/bin/khaapp-bin"

"$DAEMON_BIN" &
DAEMON_PID=$!

for i in $(seq 1 30); do
    dbus-send --session --print-reply \
        --dest=org.freedesktop.DBus \
        /org/freedesktop/DBus \
        org.freedesktop.DBus.NameHasOwner \
        string:org.khaapp.Daemon 2>/dev/null | grep -q "true" && break
    sleep 0.1
done

"$FRONTEND_BIN" "$@"
EXIT_CODE=$?

kill "$DAEMON_PID" 2>/dev/null || true
wait "$DAEMON_PID" 2>/dev/null || true

exit $EXIT_CODE
