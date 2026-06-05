# Contributing to KhaApp

Thank you for your interest in contributing.

## Before you start

- Read the legal notice in [README.md](/home/peronslayer/Desktop/KhaApp/README.md).
- Check the open issues before opening a new one.
- For significant changes, open an issue first to discuss.

## Development setup

See README.md build instructions. You need:
- Go 1.22+
- Qt 6.6+ and KDE Frameworks 6
- A real WhatsApp account to test live features

## Project structure

`daemon/` Go daemon: WhatsApp protocol, D-Bus service, SQLite
`app/` Qt6/Kirigami frontend
`dbus/` D-Bus interface XML
`packaging/` Flatpak manifest and AUR PKGBUILD

## D-Bus contract

`dbus/org.khaapp.IMessenger.xml` is the contract between daemon and frontend.
Update the XML first, then both implementations.

## Code style

- Go: `gofmt`, document exported symbols, do not ignore errors silently
- C++: Qt/KDE conventions
- QML: Kirigami patterns, user-visible strings in `qsTr()`, colors via `Kirigami.Theme.*`

## Submitting a PR

- One logical change per PR
- `cd daemon && go build ./cmd` and `cmake --build build` must pass
- If you change D-Bus methods, update the XML and both sides
- If you add user-visible strings, run `lupdate` and update both `.ts` files

## Running tests

There are currently no automated unit tests. Integration tests require a live WhatsApp account.
CI builds verify compilation only.
