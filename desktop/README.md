# asyou Desktop Client

Wails-based desktop application for managing frp tunnels with one-click setup.

## Prerequisites

- Go 1.20+
- Node.js 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

On Windows, ensure [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) is installed (pre-installed on Windows 10+).

## Development

```bash
# Start the asyou backend server first (separate terminal)
cd server && go run ./cmd/server

# Install frontend dependencies
cd desktop/frontend && npm install

# Run in Wails dev mode (from desktop/ directory)
wails dev
```

This launches the desktop app with hot-reload for the frontend.

## Production Build

```bash
cd desktop
wails build
```

The binary will be in `desktop/build/bin/asyou-desktop.exe` (Windows).

## Project Structure

```
desktop/
├── main.go              # Wails app entry
├── app.go               # App struct + Go ↔ JS bindings
├── client.go            # API client for asyou server
├── config.go            # Persistent config (JSON file)
├── discovery/           # Local port discovery
│   └── discovery.go
├── wails.json           # Wails project config
├── go.mod
└── frontend/            # React UI
    ├── package.json
    ├── src/
    │   ├── App.tsx       # Main UI (login, quick tunnel, proxy list)
    │   ├── api/bridge.ts # Wails Go bindings bridge
    │   └── index.css     # Dark theme styles
    └── index.html
```

## Features

- **One-Click Tunnel**: Select a discovered local port → auto-creates and starts a tunnel
- **Port Discovery**: Scans localhost for listening ports
- **Tunnel Management**: Start/stop tunnels, view status
- **Server Auth**: Login/register with asyou management server
- **System Tray**: Minimize to tray (via Wails)
