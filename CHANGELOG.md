# Changelog

All notable changes to KhaApp are documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

## [Unreleased]

## [0.1.0-beta.1] - 2025-01-01

### Added
- Two-process architecture: Go daemon (tulir/whatsmeow) and Qt6/Kirigami frontend
- WhatsApp multi-device protocol via QR login
- Session persistence across restarts
- Individual and group text messaging
- Image, audio and document attachments with deferred download
- Inline audio message player
- Link preview cards
- KIO-native file opening for downloaded attachments
- Unread message badges per chat and in system tray
- KDE Plasma desktop notifications with inline reply
- App lock via KWallet
- KDE Connect integration
- System tray with KStatusNotifierItem
- Read receipts
- Typing indicator
- Message search within conversations
- Lazy/paginated message loading
- Profile pictures with async cache
- Group member count display
- Keyboard shortcuts
- Settings page
- Window geometry persistence
- Kirigami color scheme reactivity
- i18n with Italian translation
- Flatpak packaging
- AUR PKGBUILD
- GitHub Actions CI and release workflows

### Known limitations
- Voice and video calls are not supported
- Status/Stories are not supported
- Live WhatsApp integration requires a real account scan
- Flathub submission is still pending

### Legal
Unofficial client. Not affiliated with Meta. May violate WhatsApp Terms of Service.
