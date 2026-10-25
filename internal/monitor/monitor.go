package monitor

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"Blaze/internal/engine"
	"Blaze/internal/store"
	"Blaze/ui"
)

// FormatBytes converts raw bytes into human-readable string (KB, MB, GB).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// StartMonitor connects to OpenVPN's management port (127.0.0.1:7505)
// and displays a real-time bandwidth speedometer and session telemetry.
func StartMonitor(ctx context.Context) error {
	state, err := store.LoadState()
	if err != nil || state == nil || !engine.ProcessAlive(state.Pid) {
		return fmt.Errorf("no active OpenVPN tunnel running. Start with 'blaze connect' first")
	}

	ui.Info(fmt.Sprintf("Connecting to OpenVPN telemetry (PID %d)...", state.Pid))

	conn, err := net.DialTimeout("tcp", "127.0.0.1:7505", 3*time.Second)
	if err != nil {
		return fmt.Errorf("could not reach OpenVPN management interface on 127.0.0.1:7505: %w", err)
	}
	defer conn.Close()

	// Request byte counter updates every 1 second
	_, err = fmt.Fprintf(conn, "bytecount 1\r\n")
	if err != nil {
		return fmt.Errorf("failed to subscribe to byte telemetry: %w", err)
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("Live Telemetry Active for %s [%s]", state.ServerName, state.Proto))
	ui.Info("Press Ctrl+C to exit monitor (tunnel remains connected).")
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	scanner := bufio.NewScanner(conn)
	var prevIn, prevOut int64
	firstSample := true

	for {
		select {
		case <-sigChan:
			fmt.Println()
			ui.Info("Exiting live telemetry monitor.")
			return nil
		default:
		}

		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		// OpenVPN outputs: >BYTECOUNT:bytes_in,bytes_out
		if strings.HasPrefix(line, ">BYTECOUNT:") {
			parts := strings.Split(strings.TrimPrefix(line, ">BYTECOUNT:"), ",")
			if len(parts) >= 2 {
				bytesIn, _ := strconv.ParseInt(parts[0], 10, 64)
				bytesOut, _ := strconv.ParseInt(parts[1], 10, 64)

				var speedIn, speedOut int64
				if !firstSample {
					speedIn = bytesIn - prevIn
					speedOut = bytesOut - prevOut
					if speedIn < 0 {
						speedIn = 0
					}
					if speedOut < 0 {
						speedOut = 0
					}
				} else {
					firstSample = false
				}

				prevIn = bytesIn
				prevOut = bytesOut

				uptime := time.Since(state.StartedAt).Round(time.Second)

				// Clear line and print live dashboard row
				fmt.Printf("\r[TELEMETRY] ⬇ %-10s (Total: %-9s) | ⬆ %-10s (Total: %-9s) | ⏱ Uptime: %-8s",
					FormatBytes(speedIn)+"/s", FormatBytes(bytesIn),
					FormatBytes(speedOut)+"/s", FormatBytes(bytesOut),
					uptime.String(),
				)
			}
		}
	}

	return nil
}
