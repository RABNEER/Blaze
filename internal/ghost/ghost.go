package ghost

import (
	"Blaze/internal/vpnbook"
	"context"
	"fmt"
)

func FlushNetworkCaches()
	dnsCmd := exec.Command("ipconfig", "/flushdns")
	dnsCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = dnsCmd.Run()

	arpCmd := exec.Command("netsh", "interface", "ip", "delete", "arpcache")
	arpCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = arpCmd.Run()

func FetchPublicIP(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil )
	if err != nil {
		return "unknown"
	}

	client := &http.client{timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "checking"
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "I dont know"
	}
	return strings.TrimSpace(string(body))
}

func PickNextServer (servers []vpnbook.Server, currentCountry string) *vpnbook.Server {
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

	ui.info("Sanitizing network traces (Flushing DNA & ARP Caches)...")
	FLushNetworkCaches()

	target := PickNextServer(server, currentCountry)
	ui.info(fmt.Sprintf("Routing Through Next Location: %s [%s]..", target.Name, target.Country))

	configPath := engine.configPath(target.ID, proto)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		ovpnData, err := vpnbook.DownloadConfig(ctx, target.Hostname, proto)
		if err != nil {

			return nil, nil, fmt.Errorf("failed to download config for %s: %w", target.Name, err)

		}
		_ = os.WriteFile(configPath, ovpnData, 0o600)

	}
	authFile, cleanupAuth, err := engine.TempAuthfile(creds.Username, creds.Password)
	 if err != nil {
		return nil, nil, fmt.Errorf("Failed to create auth file: %w", err)
	 }
	 defer cleanupAuth

	 cmd := exec.Command(ovpnBin,
			"--config", configPath,
		
			"--auth-user-pass", authFile
		
			"--auth-nocache",
		)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
		if err := cmd.Start(); err != nil {
			return nil, nil, fmt.Errorf("Failed to Start OpenVPN: %w", err)
		}
		activeState := &store.State{
			Pid:		cmd.Process.Pid,
			ServerID:	target.ID,
			ServerName:	target.Name,
			Hostname:	target.Hostname,
			Proto:		proto,
			ConfigPath:	configPath,
			StartedAt:	time.Now(),
		}
		_ = store.SaveState(activeState)

		return target, cmd, nil
}