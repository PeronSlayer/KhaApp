# KhaApp Live Integration Test Guide

This guide requires a real WhatsApp account on an Android or iPhone device. Using a secondary test account is recommended.

## Prerequisites

- KhaApp daemon and frontend built and working, or installed via Flatpak/AUR
- WhatsApp installed on a phone with the linked devices feature available
- Internet connection on both the computer and the phone

## Procedure

### 1. Start the daemon

```bash
./build/khaapp-daemon
# or: flatpak run org.khaapp.KhaApp
```

Watch for the daemon registering the D-Bus name `org.khaapp.Daemon`.

### 2. Start the frontend

```bash
./build/app/khaapp-bin
```

Skip this step when using the Flatpak launcher.

### 3. QR login

- Click `Request QR` on the login page.
- The QR image should render within 2 seconds.
- On your phone: WhatsApp, Linked Devices, Link a Device, then scan the QR.
- The frontend should transition to the chat list within 5 seconds.

If the QR does not render, check daemon logs for `QRCodeUpdated`. If the transition does not happen, check for `LoginSuccessful`.

### 4. Chat list

- Confirm recent chats appear.
- Confirm profile pictures load after a few seconds.
- Confirm unread badges match the phone app.

### 5. Send a message

- Open an individual chat.
- Type a message and press Enter or the send button.
- Confirm it appears in KhaApp and on the phone.
- Confirm the receipt changes from sent to delivered, then read.

### 6. Receive a message

- From the phone or another device, send a message to the linked account.
- Confirm it appears without manual refresh.
- Confirm a Plasma notification appears.
- Use the notification reply action and confirm the reply is sent.

### 7. Media

- Send an image from the phone, download it in KhaApp, and confirm it renders inline.
- Open the image and confirm KIO launches the system image viewer.
- Send a voice message and confirm playback controls work.

### 8. Session persistence

- Close and reopen the frontend; it should go directly to the chat list.
- Close both daemon and frontend, then reopen both; the daemon should reconnect without QR.

### 9. Network resilience

```bash
nmcli networking off
nmcli networking on
```

Confirm the daemon emits `disconnected`, then reconnects automatically when the network returns.

## Reporting Results

```text
LIVE TEST REPORT
================
QR login:               OK/FAIL - notes
Chat list:              OK/FAIL - notes
Send message:           OK/FAIL - notes
Receipt tracking:       OK/FAIL - notes
Receive + notification: OK/FAIL - notes
Notification reply:     OK/FAIL - notes
Image download:         OK/FAIL - notes
Audio playback:         OK/FAIL - notes
Session persistence:    OK/FAIL - notes
Network resilience:     OK/FAIL - notes

Daemon log errors:
```

Post the report as a GitHub issue labeled `live-test`.
