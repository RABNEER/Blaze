package main

import (
	"Blaze/internal/engine"
	"Blaze/internal/netcheck"
	"Blaze/internal/store"
	"Blaze/internal/vpnbook"
	"Blaze/ui"
	"context"
	"fmt"
	"os"
	"time"
)

func main () {
	if err := store.EnsureDirs(); err != nil {
		ui.Error(fmt.Sprintf("Failed to Initialize storage: %v", err))
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	switch os.Args[1] {
	case "servers":
		runServers()
	
	case "status"
		runStatus()

	case "disconnect"
		runDiscoonnect()

	case "Connect"
		runConnect(os.Args[2:])

	default
		ui.Warn(fmt.Sprintf("Unknown Command: %s", os.ArgArgs[1]))
		
		printUsage()
	}
}

func printUsage () {
	fmt.Println()
	ui.Info("Blaze - Fast Windows OpenVPN Cleint")
	fmt.Println("Usage: blaze <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands")
	fmt.Println("	server					List available servers with live ping latencies")
	fmt.Println("	connect [id] [flags]	Connect to a Server (or pick the fastest)")
	fmt.Println("	disconnect				Stop active OpenVPN Connection")
	fmt.Println("	status					Show Current VPN Tunnel Status")
	fmt.Println()
	fmt.Println("Connect Flags")
	fmt.Println("	--fastest				Automatically Select Server with lowest latency")
	fmt.Println("	--proto <protocol>		Protocol to use: tcp443 (default), tcp80, udp53, udp25000")
	fmt.Println()
}

func runServers()	{
	ui.Info("Fetching server list from VPNBook...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	servers, err := vpnbook.GetServers(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not fetch servers: %v", err))
		return
	}
	ui.Info(fmt.Sprintf("Pinging %d server concurretly...", len(servers)))
	
	hosts := make([]string, len(servers))
	serverMap := make(map[string] vpnbook.Server)
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

		if r.Err !=nil {
			pingStr = "timeout"
		}

		fmt.Printf("%-8s %-12s %-24s %-10s\n", s.ID, s.Country, s.Hostname, pingStr)
	}
	fmt.Println()
}

func runStatus() {
	state, err := store.LoadState()
	if err != nil || state == nil {
		ui.Info("Status: Disconnected (No active Session)")

		return 
	}

	if !engine.ProcessAlive(state.pid) {
		ui.Warn("Found Stale Session file (process is no longer running). Cleaning Up...")

		store.ClearState()
		ui.Info("Status: Disconnected")

		return
	}
	uptime := time.Since(state.StartedAt).Round(time.Second)
	ui.Success(fmt.Println("Connected to %s (%s)", state.ServerName, state.Hostname ))
	fmt.Printf("	* Server ID: %s\n", state.ServerID)
	fmt.Printf("	* Protocol:  %s\n", state.proto)
	fmt.Printf("	* PID: 		 %s\n", state.pid)
	fmt.Printf("	* Uptime: 	 %s\n", uptime)
}

