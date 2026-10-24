package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"Blaze/internal/engine"
	"Blaze/internal/ghost"
	"Blaze/internal/netcheck"
	"Blaze/internal/store"
	"Blaze/internal/vpnbook"
	"Blaze/ui"

	"github.com/0xrawsec/golang-utils/args"
	"golang.org/x/text/cases"
)

func main() {
	if err := store.EnsureDirs(); err != nil {
		ui.Error(fmt.Sprintf("Failed to initialize storage: %v", err))
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "servers":
		runServers()
	case "status":
		runStatus()
	case "disconnect", "stop":
		runDisconnect()
	case "connect":
		runConnect(os.Args[2:])
	case "hop", "ghost"
		runHop(os.Args[2:])
	default:
		ui.Warn(fmt.Sprintf("Unknown command: %s", os.Args[1]))
		printUsage()
	}
}

func printUsage() {
	fmt.Println()
	ui.Info("Blaze - Fast Windows OpenVPN Client")
	fmt.Println("Usage: blaze <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  servers                 List available servers with live ping latencies")
	fmt.Println("  connect [id] [flags]    Connect to a server (or pick the fastest)")
	fmt.Println("  disconnect              Stop active OpenVPN connection")
	fmt.Println("  status                  Show current VPN tunnel status")
	fmt.Println("  hop [flags]			   Auto-rotate IP across Countries at Set Intervals")
	fmt.Println()
	fmt.Println("Connect Flags:")
	fmt.Println("  --fastest               Automatically select server with lowest latency")
	fmt.Println("  --proto <protocol>      Protocol to use: tcp443 (default), tcp80, udp53, udp25000")
	fmt.Println()
}

func runServers() {
	ui.Info("Fetching server list from VPNBook...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	servers, err := vpnbook.GetServers(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not fetch servers: %v", err))
		return
	}
	ui.Info(fmt.Sprintf("Pinging %d servers concurrently...", len(servers)))

	hosts := make([]string, len(servers))
	serverMap := make(map[string]vpnbook.Server)
	for i, s := range servers {
		hosts[i] = s.Hostname
		serverMap[s.Hostname] = s
	}

	results := netcheck.PingAll(ctx, hosts)

	fmt.Println()
	fmt.Printf("%-8s %-12s %-24s %-10s\n", "ID", "COUNTRY", "HOSTNAME", "PING")
	fmt.Println("--------------------------------------------------------")

	for _, r := range results {
		s := serverMap[r.Hostname]
		pingStr := fmt.Sprintf("%dms", r.Latency.Milliseconds())
		if r.Err != nil {
			pingStr = "timeout"
		}
		fmt.Printf("%-8s %-12s %-24s %-10s\n", s.ID, s.Country, s.Hostname, pingStr)
	}
	fmt.Println()
}

func runStatus() {
	state, err := store.LoadState()
	if err != nil || state == nil {
		ui.Info("Status: Disconnected (no active session)")
		return
	}

	if !engine.ProcessAlive(state.Pid) {
		ui.Warn("Found stale session file (process is no longer running). Cleaning up...")
		store.ClearState()
		ui.Info("Status: Disconnected")
		return
	}

	uptime := time.Since(state.StartedAt).Round(time.Second)
	ui.Success(fmt.Sprintf("Connected to %s (%s)", state.ServerName, state.Hostname))
	fmt.Printf("  • Server ID: %s\n", state.ServerID)
	fmt.Printf("  • Protocol:  %s\n", state.Proto)
	fmt.Printf("  • PID:       %d\n", state.Pid)
	fmt.Printf("  • Uptime:    %s\n", uptime)
}

func runDisconnect() {
	state, err := store.LoadState()
	if err != nil || state == nil {
		ui.Info("No active VPN session found.")
		return
	}

	ui.Info(fmt.Sprintf("Terminating OpenVPN process (PID %d)...", state.Pid))
	if engine.ProcessAlive(state.Pid) {
		if err := engine.KillProcess(state.Pid); err != nil {
			ui.Error(fmt.Sprintf("Failed to kill process: %v", err))
			return
		}
	}
	store.ClearState()
	ui.Success("Disconnected successfully.")
}

func runConnect(args []string) {
	// 1. Check Administrator Privileges
	if !engine.IsAdmin() {
		ui.Warn("OpenVPN requires Administrator privileges to configure routing.")
		ui.Info("Requesting Windows UAC elevation...")
		if err := engine.SelfElevate(os.Args[1:]); err != nil {
			ui.Error(fmt.Sprintf("UAC elevation failed: %v", err))
		}
		return
	}

	// 2. Parse flags
	fs := flag.NewFlagSet("connect", flag.ExitOnError)
	fastest := fs.Bool("fastest", false, "Connect to the lowest latency server")
	proto := fs.String("proto", "tcp443", "Protocol (tcp443, tcp80, udp53, udp25000)")
	_ = fs.Parse(args)

	if !vpnbook.ValidateProtocol(*proto) {
		ui.Error(fmt.Sprintf("Invalid protocol '%s'. Valid options: %v", *proto, vpnbook.Protocols))
		return
	}

	// 3. Find OpenVPN binary
	ovpnBin, found := engine.FindBinary()
	if !found {
		ui.Error("openvpn.exe not found! Please install OpenVPN.")
		return
	}

	// 4. Check if already connected
	if state, err := store.LoadState(); err == nil && state != nil {
		if engine.ProcessAlive(state.Pid) {
			ui.Warn(fmt.Sprintf("Already connected to %s (PID %d). Run 'blaze disconnect' first.", state.ServerName, state.Pid))
			return
		}
		store.ClearState()
	}

	ctx := context.Background()

	// 5. Fetch server list
	servers, err := vpnbook.GetServers(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to get server list: %v", err))
		return
	}

	var target *vpnbook.Server
	targetID := ""
	if fs.NArg() > 0 {
		targetID = fs.Arg(0)
	}

	if *fastest || targetID == "" {
		ui.Info("Benchmarking servers to find the fastest...")
		hosts := make([]string, len(servers))
		for i, s := range servers {
			hosts[i] = s.Hostname
		}

		ranked := netcheck.PingAll(ctx, hosts)
		if len(ranked) == 0 || ranked[0].Err != nil {
			ui.Warn("Ping benchmark failed. Using default server.")
			target = &servers[0]
		} else {
			target = vpnbook.FindServer(servers, ranked[0].Hostname)
			ui.Success(fmt.Sprintf("Fastest server: %s (%dms)", target.Name, ranked[0].Latency.Milliseconds()))
		}
	} else {
		target = vpnbook.FindServer(servers, targetID)
		if target == nil {
			ui.Error(fmt.Sprintf("Server '%s' not found. Run 'blaze servers' to see valid IDs.", targetID))
			return
		}
	}

	// 6. Fetch live rotating credentials
	ui.Info("Fetching live credentials from VPNBook...")
	creds, err := vpnbook.GetCredentials(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not fetch credentials: %v", err))
		return
	}

	// 7. Ensure .ovpn config is downloaded and cached
	configPath := engine.ConfigCheckPath(target.ID, *proto)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		ui.Info(fmt.Sprintf("Downloading configuration for %s (%s)...", target.Name, *proto))
		ovpnData, err := vpnbook.DownloadConfig(ctx, target.Hostname, *proto)
		if err != nil {
			ui.Error(fmt.Sprintf("Failed to download config: %v", err))
			return
		}
		if err := os.WriteFile(configPath, ovpnData, 0o600); err != nil {
			ui.Error(fmt.Sprintf("Failed to save config: %v", err))
			return
		}
	}

	// 8. Create temporary credential file and defer its deletion
	authFile, cleanupAuth, err := engine.TempAuthFile(creds.Username, creds.Password)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create temporary auth file: %v", err))
		return
	}
	defer cleanupAuth()

	// 9. Launch OpenVPN process
	ui.Info(fmt.Sprintf("Starting tunnel to %s [%s]...", target.Name, *proto))
	cmd := exec.Command(ovpnBin,
		"--config", configPath,
		"--auth-user-pass", authFile,
		"--auth-nocache",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}

	if err := cmd.Start(); err != nil {
		ui.Error(fmt.Sprintf("Failed to launch OpenVPN: %v", err))
		return
	}

	// 10. Record session in state.json
	activeState := &store.State{
		Pid:        cmd.Process.Pid,
		ServerID:   target.ID,
		ServerName: target.Name,
		Hostname:   target.Hostname,
		Proto:      *proto,
		ConfigPath: configPath,
		StartedAt:  time.Now(),
	}
	_ = store.SaveState(activeState)

	ui.Success(fmt.Sprintf("Tunnel process started! PID: %d", cmd.Process.Pid))
	ui.Info("Press Ctrl+C to disconnect...")

	// 11. Listen for Ctrl+C or process termination using channels
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	doneChan := make(chan error, 1)
	go func() {
		doneChan <- cmd.Wait()
	}()

	select {
	case <-sigChan:
		fmt.Println()
		ui.Info("Received interrupt signal. Stopping OpenVPN...")
		_ = engine.KillProcess(cmd.Process.Pid)
		store.ClearState()
		ui.Success("Disconnected.")
	case err := <-doneChan:
		store.ClearState()
		if err != nil {
			ui.Error(fmt.Sprintf("OpenVPN process ended with error: %v", err))
		} else {
			ui.Info("OpenVPN process closed.")
		}
	}
}

func runHop(args []string) {
	if !engine.IsAdmin() {
		ui.Warn("Ghost Hop requires Administrator Priviles to manage routing.")
		ui.Warn("Requesting Windows UAC elevation...")

		if err := engine.SelfElevate(os.Args[1:]); err != nil {
			ui.Error(fmt.Sprintf("UAC Elevation Failed: %v", err))
		}
		return
	}
	fs := flag.NewFlagSet("hop", flag.ExitOnError)
	Interval := fs.Duration("every", 3*time.Minute, "Rotation interval (e.g. 1m, 3m, 5m)")
	_ = fs.Parse(args)

	
	ovpnBin, found := engine.FindBinary()
		if !found {
			ui.Error("openvpn.exe not found! Please Install OpenVPN.")

		return
		
	}

	ctx := context.Background()

	ui.Info("Initialize Ghost Protocol....")
	servers, err := vpnbook.GetServers(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to fetch servers: %v", err))

		return
	}

	creds, err :=  vpnbook.GetCredentials(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to fetch credentials: %v", err))

		return
	}

	fmt.Println()
	ui.Info(fmt.Sprintf("Ghost Protocol is Active - Rotating IP every %v", *interval))
	ui.Info("Press Ctrl+C to terminate Ghost Mode and Close tunnel")
	fmt.Println()

	currentCountry := ""
	var currentCmd *exec.Cmd

	cleanup := func() {
		if currentCmd != nil && currentCmd.Process != nil {
			_= engine.KillProcess(currentCmd.Process.Pid)
		}

		store.ClearState()
		ghost.FlushNetworkCaches()
		ui.Success("Ghost Protocol deactivated. Identity Wiped")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	doHop := func() {
		target, cmd, err := ghost.ExecuteHop(ctx, *proto, currentCountry, servers, creds, ovpnBin)

		if err != nil {
			ui.Error(fmt.Sprintf("Hop Failed: %v", err))
			
			return
		}

		currentCmd = cmd 

		currentCountry = target.Country

		time.Sleep(5 * time.Second)

		newIP := ghost.FetchPublicIP(ctx)

		ui.Succes(fmt.Sprintf("Identity Active: %s (%s) | Verified IP: %s", target.Name, target.Country, &newIP))

		ui.Info(fmt.Sprintf("Next hop in %v...", *interval))

		fmt.Println()
	}

	doHop()


	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println()
				ui.Warn("Interrupt received. Disconnecting...")
				cleanup()
				return
			case <-ticker.C:
				ui.Info("Hop Interval Reached. Rotating Identity...")
				doHop()
			
		}
	}
}