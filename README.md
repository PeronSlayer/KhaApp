# KhaApp

KhaApp is an unofficial, open-source, native WhatsApp client for KDE Plasma on Linux. It is not affiliated with, endorsed by, or supported by Meta or WhatsApp.

## Installation

### Flatpak (recommended)

```bash
flatpak install flathub org.khaapp.KhaApp
flatpak run org.khaapp.KhaApp
```

*(Flathub submission pending — build locally in the meantime, see below)*

### AUR (Arch / CachyOS / Manjaro)

```bash
yay -S khaapp
# or
paru -S khaapp
```

### Build from source

**Dependencies (Arch/CachyOS):**

```bash
sudo pacman -S go cmake extra-cmake-modules ninja base-devel pkg-config \
  sqlite qt6-base qt6-declarative qt6-multimedia \
  kirigami knotifications kwallet kf6-kio
```

**Dependencies (Ubuntu 24.04+):**

```bash
sudo apt-get install go cmake ninja-build extra-cmake-modules pkg-config \
  libsqlite3-dev qt6-base-dev qt6-declarative-dev qt6-multimedia-dev \
  libkf6kirigami2-dev libkf6notifications-dev libkf6wallet-dev libkf6kio-dev
```

**Build:**

```bash
git clone https://github.com/khaapp/khaapp.git
cd khaapp
cd daemon && go mod vendor && CGO_ENABLED=1 go build -o ../build/khaapp-daemon ./cmd && cd ..
cmake -B build -S . -GNinja -DCMAKE_BUILD_TYPE=Release
cmake --build build --parallel
```

**Run:**

```bash
# Terminal 1:
./build/khaapp-daemon
# Terminal 2:
./build/app/khaapp-bin
```

Or use the launcher wrapper (starts both automatically):

```bash
./packaging/aur/khaapp-native-launcher.sh
```

## Architecture

KhaApp uses a two-process model. The Go daemon owns the WhatsApp protocol connection and exposes a D-Bus service, while the Qt6/Kirigami frontend consumes that service over the session bus and never talks to WhatsApp directly.

## Legal notice

KhaApp is an unofficial, third-party client for WhatsApp.
It is not affiliated with, endorsed by, or connected to Meta Platforms, Inc.
Using unofficial WhatsApp clients may violate WhatsApp's Terms of Service.
Use at your own risk.
