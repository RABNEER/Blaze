# Blaze ⚡

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows%2010%20%2F%2011-blue?style=flat&logo=windows)](https://microsoft.com/windows)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Security](https://img.shields.io/badge/Security-Cryptographic%20Zero--Trace-red.svg)](README.md)

A fast, lightweight, and modern CLI OpenVPN client & cybersecurity suite for Windows written in Go. Blaze automates fetching free high-speed OpenVPN servers and credentials from VPNBook, measures network latencies in parallel, and equips your machine with advanced anti-tracking, anti-censorship, and emergency privacy protocols.

---

## 🛡️ Core Cybersecurity Weapons

| Feature | Command | Description |
|---|---|---|
| **Tunnel Engine** | `blaze connect [id]` | Automated scraping, UAC self-elevation, config caching, and process supervision. |
| **Latency Benchmark** | `blaze servers` | Concurrent TCP pinging across all candidate servers using Go goroutines. |
| **Ghost Protocol** | `blaze hop --every 3m` | Auto-rotates your public IP across different countries, sanitizing DNS/ARP caches between hops. |
| **Leak Hunter** | `blaze audit` | Active penetration testing for IPv6 exposure and Windows Smart Multi-Homed DNS hijacking. |
| **Cyber Shield** | `blaze shield [--on\|--off]` | 1-click hardening: disables IPv6 stack and enforces encrypted Cloudflare/Quad9 DNS. |
| **DNS Sinkhole** | `blaze block` | Zero-dependency RFC-1035 UDP DNS server (`127.0.0.1:53`) that drops 50+ ad/telemetry trackers (`0.0.0.0`). |
| **Hardware Kill-Switch** | `blaze killswitch [--on\|--off]` | Windows Firewall isolation that air-gaps all non-VPN outbound traffic to prevent unencrypted leaks. |
| **Live Speedometer** | `blaze monitor` | Streams live download/upload speeds and total session volume via OpenVPN's TCP management interface. |
| **MAC Spoofer** | `blaze spoof [--reset]` | Generates IEEE 802-compliant locally-administered unicast MAC addresses to prevent Wi-Fi hardware tracking. |
| **Emergency Panic** | `blaze panic` | Instantly drops tunnels, cryptographically shreds configs/creds (`crypto/rand`), flushes network tables, and restores settings. |

---

## 📁 Project Architecture

```text
Blaze/
├── internal/
│   ├── dnsfilter/     # RFC-1035 UDP DNS sinkhole & ad/tracker blocklist
│   ├── engine/        # OpenVPN binary locator, UAC elevation, PID lifecycle
│   ├── ghost/         # Ghost Hop protocol (multi-country IP rotation & cache wiping)
│   ├── killswitch/    # Windows Advanced Firewall outbound kill-switch
│   ├── monitor/       # OpenVPN TCP management socket telemetry & speedometer
│   ├── netcheck/      # Concurrent TCP pinging and server latency ranking
│   ├── panic/         # Cryptographic file shredding and emergency zero-trace wiping
│   ├── shield/        # Leak detection (IPv6 / DNS) and Windows adapter hardening
│   ├── spoof/         # IEEE 802 MAC address generator & network adapter cycler
│   ├── store/         # Persistent state, credentials, and configs (%LOCALAPPDATA%\BlazeVPN)
│   └── vpnbook/       # VPNBook scraper, config downloader, and server directory
├── ui/                # ANSI terminal styling and status indicators
├── main.go            # Subcommand CLI router
├── go.mod             # Go module definition
└── README.md          # Project documentation
```

---

## 🚀 Quick Start & CLI Reference

### Prerequisites
1. **Go 1.21+** installed ([go.dev/dl](https://go.dev/dl/)).
2. **OpenVPN for Windows** installed (default path: `C:\Program Files\OpenVPN\bin\openvpn.exe` or on system `PATH`).
3. **Administrator Privileges**: OpenVPN, firewall rules, and adapter reconfiguration require elevation. Blaze handles this automatically via UAC self-elevation.

### 1. Build and Run Tests

```powershell
# Run the complete unit test suite
go test -v ./...

# Compile the standalone Windows executable
go build -o blaze.exe .
```

### 2. Basic Tunnel Operations

```powershell
# Benchmark all candidate endpoints with live ping latencies
.\blaze.exe servers

# Connect to the lowest-latency server automatically
.\blaze.exe connect --fastest

# Connect to a specific server and protocol
.\blaze.exe connect ca149 --proto tcp443
# Supported protocols: tcp443 (default), tcp80, udp53, udp25000

# View active session uptime, PID, and server metadata
.\blaze.exe status

# Disconnect the active tunnel
.\blaze.exe disconnect
```

### 3. Advanced Cybersecurity Commands

```powershell
# Auto-rotate IP every 3 minutes across different countries
.\blaze.exe hop --every 3m

# Run an active penetration test for IPv6 and DNS leaks
.\blaze.exe audit

# Harden your network against leaks (disable IPv6, enforce Cloudflare/Quad9 DNS)
.\blaze.exe shield --on

# Launch the local UDP DNS sinkhole (blocks ads and telemetry in real time)
.\blaze.exe block

# Arm the hardware firewall kill-switch (blocks all non-VPN outbound traffic)
.\blaze.exe killswitch --on

# Disarm the kill-switch and restore normal traffic
.\blaze.exe killswitch --off

# Stream live download/upload speeds and data telemetry
.\blaze.exe monitor

# Randomize your physical network card's hardware MAC address
.\blaze.exe spoof

# Restore original factory hardware MAC address
.\blaze.exe spoof --reset

# EMERGENCY PANIC: Drop tunnel and cryptographically shred all traces on disk
.\blaze.exe panic
```

---

## 🔒 Security & Privacy Mechanics

### 1. Cryptographic File Shredding
Standard file deletion (`os.Remove`) only unlinks the file metadata, leaving the actual file blocks intact on your storage drive where undelete tools (e.g. Recuva) can recover them. The `blaze panic` engine calculates the exact file size, overwrites all bytes with cryptographically secure pseudo-random noise (`crypto/rand`), forces the write to disk, and only then removes the file.

### 2. Multi-Homed DNS Hijack Prevention
Windows includes a feature called *Smart Multi-Homed Name Resolution* that simultaneously queries every configured network interface (including your home ISP router) to resolve domain names as quickly as possible. `blaze audit` detects this behavior, while `blaze shield` enforces strict adapter-level DNS bindings to prevent queries from escaping unencrypted.

### 3. IEEE 802 MAC Address Compliance
Wi-Fi hardware access control lists and router drivers require MAC addresses to have the **Locally Administered** bit set (bit 1 of the first octet = 1) and the **Unicast** bit set (bit 0 = 0). `blaze spoof` enforces this standard (second nibble restricted to `2`, `6`, `A`, or `E`) to ensure maximum driver compatibility across Realtek, Intel, and Qualcomm network chipsets.

---

## 🛠️ Troubleshooting

- **`openvpn.exe not found`**: Ensure OpenVPN is installed from [openvpn.net](https://openvpn.net/community-downloads/). Blaze checks `C:\Program Files\OpenVPN\bin\openvpn.exe`, `C:\Program Files (x86)\OpenVPN\bin\openvpn.exe`, and your system `PATH`.
- **`failed to bind UDP port 53`**: Windows Internet Connection Sharing (ICS) or another local DNS resolver may be occupying port 53. Run the shell as Administrator, or stop conflicting services with `Stop-Service SharedAccess`.
- **UAC Elevation Prompts**: Modifying Windows routing tables, adapter bindings, and firewall rules requires administrator rights. When prompted by Windows User Account Control, select **Yes** to allow Blaze to apply the configuration.

---

## License

This project is licensed under the [MIT License](LICENSE).
