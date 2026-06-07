#!/bin/bash
# Relocatable launcher for the KhaApp portable bundle.
# Starts the daemon, waits for its D-Bus name, then runs the frontend.
# Binaries are expected to live next to this script.

set -u

HERE="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"

DAEMON="$HERE/khaapp-daemon"
FRONTEND="$HERE/khaapp-bin"

if [ ! -x "$DAEMON" ] || [ ! -x "$FRONTEND" ]; then
    echo "khaapp: missing khaapp-daemon or khaapp-bin next to launcher" >&2
    exit 1
fi

"$DAEMON" &
DAEMON_PID=$!

cleanup() {
    kill "$DAEMON_PID" 2>/dev/null || true
    wait "$DAEMON_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

for _ in $(seq 1 50); do
    if dbus-send --session --print-reply \
        --dest=org.freedesktop.DBus \
        /org/freedesktop/DBus \
        org.freedesktop.DBus.NameHasOwner \
        string:org.khaapp.Daemon 2>/dev/null | grep -q "true"; then
        break
    fi
    sleep 0.1
done

"$FRONTEND" "$@"
exit $?
