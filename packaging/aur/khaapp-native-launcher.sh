#!/bin/bash

/usr/bin/khaapp-daemon &
DAEMON_PID=$!

for i in $(seq 1 30); do
    dbus-send --session --print-reply \
        --dest=org.freedesktop.DBus \
        /org/freedesktop/DBus \
        org.freedesktop.DBus.NameHasOwner \
        string:org.khaapp.Daemon 2>/dev/null | grep -q "true" && break
    sleep 0.1
done

/usr/bin/khaapp-bin "$@"
EXIT_CODE=$?

kill "$DAEMON_PID" 2>/dev/null || true
wait "$DAEMON_PID" 2>/dev/null || true

exit $EXIT_CODE
