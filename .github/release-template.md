# Relay Switch v{{VERSION}}

## Downloads

Choose the delivery path that matches your environment.

### Desktop

Use the packaged desktop app if you want the full Electron experience on macOS, Windows, or Linux desktop systems.

### macOS

1. Apple Silicon:
   - `Relay-Switch-{{VERSION}}-arm64.pkg`
   - `Relay-Switch-{{VERSION}}-arm64.dmg`
   - `Relay-Switch-{{VERSION}}-mac-arm64.zip`
2. Intel Mac:
   - use the matching `x64` artifact when that build is attached

### Windows

1. Installer:
   - `Relay-Switch-{{VERSION}}-x64-setup.exe`
2. If an `arm64` installer is attached, prefer that on Windows on ARM devices

### Linux

1. `Relay-Switch-{{VERSION}}-x64.AppImage`
2. `Relay-Switch-{{VERSION}}-linux-x64.tar.gz`

### WSL / Linux Server

Use the server package if you want browser-based management on `WSL` or a plain `Linux server`.

1. `relay-switch-server_{{VERSION}}_linux_amd64.tar.gz`
2. `relay-switch-server_{{VERSION}}_linux_arm64.tar.gz`
3. `relay-switch-server_{{VERSION}}_SHA256SUMS.txt`

## Install Paths

### Desktop install

Download the desktop package that matches your OS and launch it normally.

### Server install

Latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | bash
```

Pinned release:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | RELAY_SWITCH_VERSION=vX.Y.Z bash
```

Development-only source install:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install-from-source.sh | bash
```

## Install Notes

### macOS

This build is signed with Developer ID and intended to be distributed with Apple notarization.

If macOS blocks the app on first launch:

1. Right click the app and choose `Open`
2. Or go to `System Settings -> Privacy & Security` and allow the app to open
3. Prefer the `.pkg` installer or move the `.app` into `/Applications` before launch

### Windows

This build is currently unsigned.

If SmartScreen warns that the publisher is unknown:

1. Click `More info`
2. Then click `Run anyway`

### Linux

For AppImage:

```bash
chmod +x "Relay-Switch-{{VERSION}}-x64.AppImage"
./Relay-Switch-{{VERSION}}-x64.AppImage
```

## Notes

1. The desktop app includes the local `relay-switch-core` binary. Users do not need to install Go.
2. Automatic updates are only available in packaged builds.
3. Provider credentials remain local to the device.
4. The production server installer uses GitHub Release assets by default.
5. `scripts/install-from-source.sh` is intended for development and validation, not for production deployment.

## Verification Checklist

1. Desktop users can identify the correct package for their OS.
2. Server users can identify the correct release asset or installer command.
3. `core` status becomes running after launch or install.
4. Local API base is shown in the app or Web UI.
5. Provider health checks succeed.

## Changelog

- Fill in user-visible changes here
