# Blaze ⚡

A fast, lightweight, and modern CLI OpenVPN client for Windows written in Go. Blaze automates fetching free, high-speed OpenVPN servers and credentials from VPNBook, measures network latencies in parallel, and manages your tunnel directly from the terminal.

---

## Features

- **Automated Credential & Server Fetching**: Automatically scrapes live credentials and OpenVPN configs from VPNBook with built-in fallbacks.
- **Concurrent TCP Latency Probing**: Pings candidate servers concurrently using Go goroutines to select the fastest server.
- **Windows System Integration**: Auto-detects `openvpn.exe`, handles UAC self-elevation (`RunAs`), and provides clean process tracking (`STILL_ACTIVE` monitoring and termination).
- **Persistent Local Cache**: Saves session states, credentials, and `.ovpn` configuration files in `%LOCALAPPDATA%\BlazeVPN`.
- **Colored CLI Output**: Terminal interface with ANSI styling and clear visual status indicators.

---

## Project Structure

```text
Blaze/
├── internal/
│   ├── engine/        # OpenVPN binary detection, Windows UAC elevation, PID management
│   ├── netcheck/      # Concurrent TCP pinging and server latency sorting
│   ├── store/         # State and credential file persistence (%LOCALAPPDATA%\BlazeVPN)
│   └── vpnbook/       # VPNBook scraper, config downloader, and server directory
├── ui/                # Terminal output formatting and ANSI color helpers
├── go.mod             # Go module definition
└── README.md          # Project documentation
```

---

## Prerequisites

1. **Go**: Version `1.21+` installed ([golang.org](https://go.dev/dl/)).
2. **OpenVPN**: Installed on Windows (default install path: `C:\Program Files\OpenVPN\bin\openvpn.exe` or available in your system `PATH`).
3. **Administrator Privileges**: OpenVPN requires administrative rights on Windows to configure network TAP/TUN adapters and modify routing tables. Blaze includes automated UAC elevation helpers.

---

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/<your-username>/Blaze.git
cd Blaze
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Verify Build

Compile all packages across the workspace:

```bash
go build ./...
```

---

## Package Overview

### `internal/engine`
- `FindBinary()`: Scans `%ProgramFiles%` and `PATH` to locate the OpenVPN binary.
- `IsAdmin()` / `SelfElevate()`: Inspects Windows access tokens and restarts with elevated privileges when necessary.
- `ProcessAlive(pid)` & `KillProcess(pid)`: Tracks and terminates running OpenVPN instances using native Windows process handles.
- `TempAuthFile(user, pass)`: Generates disposable credential files for OpenVPN authentication.

### `internal/vpnbook`
- `GetServers(ctx)`: Parses active VPNBook endpoints with fallback servers for US, CA, UK, DE, and FR.
- `GetCredentials(ctx)`: Scrapes active VPN credentials (username & dynamically rotating password).
- `DownloadConfig(ctx, host, proto)`: Downloads `.ovpn` profiles for supported protocols (`tcp443`, `tcp80`, `udp53`, `udp25000`).

### `internal/netcheck`
- `TCPPing(ctx, host, port, timeout)`: Measures round-trip connection time.
- `PingAll(ctx, hosts)`: Concurrently measures latencies across all servers and ranks them fastest to slowest.

### `internal/store`
- Manages `%LOCALAPPDATA%\BlazeVPN`:
  - `state.json`: Active connection metadata (PID, server ID, start timestamp).
  - `creds.json`: Cached credentials with timestamps.
  - `configs/`: Downloaded OpenVPN configuration profiles.

### `ui`
- Terminal formatting utilities (`Success`, `Error`, `Warn`, `Info`, `Dimf`) with ANSI color escapes.

---

## License

This project is licensed under the [MIT License](LICENSE).
