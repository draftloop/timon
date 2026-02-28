#!/bin/sh
set -e

REPO="draftloop/timon"
BIN="timon"
INSTALL_DIR="/usr/local/bin"

# Require root
if [ "$(id -u)" -ne 0 ]; then
  echo "This script must be run as root." && exit 1
fi

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "Unsupported OS: $OS" && exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *)               echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# Fetch latest version tag
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version" && exit 1
fi

FILENAME="${BIN}_${OS}_${ARCH}"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

# Check installed version
if command -v "$BIN" > /dev/null 2>&1; then
  INSTALLED=$(${BIN} version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
  if [ "$INSTALLED" = "$(echo "$VERSION" | tr -d 'v')" ]; then
    echo "$BIN $VERSION is already installed." && exit 0
  fi
  if [ -n "$INSTALLED" ]; then
    # Compare versions: strip 'v' and split into parts
    installed_parts=$(echo "$INSTALLED" | tr -d 'v' | tr '.' ' ')
    new_parts=$(echo "$VERSION" | tr -d 'v' | tr '.' ' ')
    set -- $installed_parts; i1=$1 i2=$2 i3=$3
    set -- $new_parts;       n1=$1 n2=$2 n3=$3
    if [ "$n1" -lt "$i1" ] || { [ "$n1" -eq "$i1" ] && [ "$n2" -lt "$i2" ]; } || { [ "$n1" -eq "$i1" ] && [ "$n2" -eq "$i2" ] && [ "$n3" -lt "$i3" ]; }; then
      echo "Cannot downgrade: installed $INSTALLED → requested $VERSION." && exit 1
    fi
  fi
fi

# Stop running timon if any
if pgrep -x "$BIN" > /dev/null 2>&1; then
  CURRENT_VERSION=$($BIN version 2>/dev/null || echo "unknown")
  printf "Timon is running (current: %s, installing: %s). Stop it before installing? [y/N] " "$CURRENT_VERSION" "$VERSION"
  read -r answer < /dev/tty
  case "$answer" in
    y|Y)
      if command -v rc-service > /dev/null 2>&1 && rc-service timon status > /dev/null 2>&1; then
        rc-service timon stop || true
      elif command -v systemctl > /dev/null 2>&1; then
        systemctl --user stop timon.service 2>/dev/null || true
      else
        pkill -x "$BIN" || true
      fi
      echo "Timon stopped."
      ;;
    *)
      echo "Aborted." && exit 1
      ;;
  esac
fi

echo "Installing $BIN $VERSION ($OS/$ARCH)..."

TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP"
chmod +x "$TMP"
mv "$TMP" "$INSTALL_DIR/$BIN"

echo "$BIN installed to $INSTALL_DIR/$BIN"
$BIN version

# Install service
case "$OS" in
  linux)
    if command -v systemctl > /dev/null 2>&1; then
      cat > /etc/systemd/system/timon.service <<EOF
[Unit]
Description=Timon daemon
After=network.target

[Service]
ExecStart=$INSTALL_DIR/$BIN daemon --background
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
      systemctl daemon-reload
      systemctl enable --now timon.service
      echo "Timon service enabled and started (systemd system service)"
    elif command -v rc-update > /dev/null 2>&1; then
      cat > /etc/init.d/timon <<EOF
#!/sbin/openrc-run
description="Timon daemon"
command="$INSTALL_DIR/$BIN"
command_args="daemon --background"
command_background=true
pidfile="/run/timon.pid"
EOF
      chmod +x /etc/init.d/timon
      rc-update add timon default
      rc-service timon start
      echo "Timon service enabled and started (OpenRC)"
    else
      echo "Warning: no supported init system found (systemd or OpenRC). Start Timon manually: $INSTALL_DIR/$BIN daemon --background"
    fi
    ;;
  darwin)
    PLIST_DIR="$HOME/Library/LaunchAgents"
    mkdir -p "$PLIST_DIR"
    cat > "$PLIST_DIR/com.timon.daemon.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.timon.daemon</string>
  <key>ProgramArguments</key>
  <array>
    <string>$INSTALL_DIR/$BIN</string>
    <string>daemon</string>
    <string>--background</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
EOF
    launchctl load -w "$PLIST_DIR/com.timon.daemon.plist"
    echo "timon service loaded (launchd LaunchAgent)"
    ;;
esac