# Clash for AI v{{VERSION}}

## Downloads

Choose the package that matches your desktop system:

### macOS

1. Apple Silicon:
   - `Clash-for-AI-{{VERSION}}-arm64.pkg`
   - `Clash-for-AI-{{VERSION}}-arm64.dmg`
   - `Clash-for-AI-{{VERSION}}-mac-arm64.zip`
2. Intel Mac:
   - use the matching `x64` artifact when that build is attached

### Windows

1. Installer:
   - `Clash-for-AI-{{VERSION}}-x64-setup.exe`
2. If an `arm64` installer is attached, prefer that on Windows on ARM devices

### Linux

1. `Clash-for-AI-{{VERSION}}-x64.AppImage`
2. `Clash-for-AI-{{VERSION}}-linux-x64.tar.gz`

## Install Notes

### macOS

This build is currently distributed without Apple notarization.

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
chmod +x "Clash-for-AI-{{VERSION}}-x64.AppImage"
./Clash-for-AI-{{VERSION}}-x64.AppImage
```

## Notes

1. The desktop app includes the local `clash-for-ai-core` binary. Users do not need to install Go.
2. Automatic updates are only available in packaged builds.
3. Provider credentials remain local to the device.

## Verification Checklist

1. App launches successfully
2. `core` status becomes running
3. Local API base is shown in the app
4. Provider health checks succeed

## Changelog

- Fill in user-visible changes here
