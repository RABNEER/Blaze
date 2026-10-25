package ghost

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"Blaze/internal/engine"
	"Blaze/internal/store"
	"Blaze/internal/vpnbook"
	"Blaze/ui"
)

func FlushNetworkCaches() {
	dnsCmd := exec.Command("ipconfig", "/flushdns")
	dnsCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = dnsCmd.Run()

	arpCmd := exec.Command("netsh", "interface", "ip", "delete", "arpcache")
	arpCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = arpCmd.Run()
}

func FetchPublicIP(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	if err != nil {
		return "unknown"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "checking..."
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(body))
}

func PickNextServer(servers []vpnbook.Server, currentCountry string) *vpnbook.Server {
	var candidates []vpnbook.Server

	for _, s := range servers {
		if s.Country != currentCountry {
			candidates = append(candidates, s)
		}
	}

	if len(candidates) == 0 {
		candidates = servers
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	idx := r.Intn(len(candidates))
	return &candidates[idx]
}

func ExecuteHop(ctx context.Context, proto string, currentCountry string, servers []vpnbook.Server, creds *vpnbook.Credentials, ovpnBin string) (*vpnbook.Server, *exec.Cmd, error) {
	if state, err := store.LoadState(); err == nil && state != nil {
		if engine.ProcessAlive(state.Pid) {
			_ = engine.KillProcess(state.Pid)
		}
		store.ClearState()
	}

	ui.Info("Sanitizing network traces (Flushing DNS & ARP caches)...")
	FlushNetworkCaches()

	target := PickNextServer(servers, currentCountry)
	ui.Info(fmt.Sprintf("Routing through next location: %s [%s]...", target.Name, target.Country))

	configPath := engine.ConfigCheckPath(target.ID, proto)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		ovpnData, err := vpnbook.DownloadConfig(ctx, target.Hostname, proto)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to download config for %s: %w", target.Name, err)
		}
		_ = os.WriteFile(configPath, ovpnData, 0o600)
	}

	authFile, cleanupAuth, err := engine.TempAuthFile(creds.Username, creds.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create auth file: %w", err)
	}
	defer cleanupAuth()

	cmd := exec.Command(ovpnBin,
		"--config", configPath,
		"--auth-user-pass", authFile,
		"--auth-nocache",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start OpenVPN: %w", err)
	}

	activeState := &store.State{
		Pid:        cmd.Process.Pid,
		ServerID:   target.ID,
		ServerName: target.Name,
		Hostname:   target.Hostname,
		Proto:      proto,
		ConfigPath: configPath,
		StartedAt:  time.Now(),
	}
	_ = store.SaveState(activeState)

	return target, cmd, nil
}
