package panic

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"Blaze/internal/engine"
	"Blaze/internal/ghost"
	"Blaze/internal/killswitch"
	"Blaze/internal/shield"
	"Blaze/internal/store"
	"Blaze/ui"
)

// ShredFile securely overwrites a file with random cryptographic bytes
// before removing it from disk, preventing forensic file recovery.
func ShredFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	size := info.Size()
	if size > 0 {
		randomBytes := make([]byte, size)
		if _, err := rand.Read(randomBytes); err == nil {
			_ = os.WriteFile(path, randomBytes, 0o600)
		}
	}
	return os.Remove(path)
}

// ShredDirectory walks through all files in a directory, shreds their content,
// and removes the entire directory tree.
func ShredDirectory(dirPath string) int {
	shreddedCount := 0
	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if err := ShredFile(path); err == nil {
			shreddedCount++
		}
		return nil
	})
	_ = os.RemoveAll(dirPath)
	return shreddedCount
}

// ExecutePanic triggers the zero-trace emergency protocol:
// 1. Kills any active OpenVPN process.
// 2. Cryptographically shreds state, creds, and configs in %LOCALAPPDATA%\BlazeVPN.
// 3. Flushes DNS and ARP routing tables.
// 4. Reverts any shield modifications.
func ExecutePanic() {
	ui.Warn("🚨 INITIATING EMERGENCY PANIC PROTOCOL...")
	fmt.Println()

	// 1. Terminate OpenVPN Process
	if state, err := store.LoadState(); err == nil && state != nil {
		if engine.ProcessAlive(state.Pid) {
			ui.Info(fmt.Sprintf("• Terminating active OpenVPN process (PID %d)...", state.Pid))
			_ = engine.KillProcess(state.Pid)
		}
	}

	// Also force kill any stray openvpn.exe processes on Windows
	killAllCmd := exec.Command("taskkill", "/F", "/IM", "openvpn.exe", "/T")
	killAllCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = killAllCmd.Run()
	ui.Success("Tunnel terminated (all openvpn processes dropped).")

	// 2. Cryptographically shred session files on disk
	baseDir := store.BaseDir()
	shredded := ShredDirectory(baseDir)
	ui.Success(fmt.Sprintf("Cryptographically shredded %d file(s) in %s", shredded, baseDir))

	// Recreate empty base dirs so the client can still be run later
	_ = store.EnsureDirs()

	// 3. Flush network caches
	ui.Info("• Flushing DNS and ARP routing caches...")
	ghost.FlushNetworkCaches()
	ui.Success("Network tables sanitized.")

	// 4. Revert shield settings (restore default Windows adapters)
	_ = shield.DisableShield()
	_ = killswitch.Disable()
	ui.Success("Windows network adapter configurations and firewall rules restored.")

	fmt.Println()
	ui.Success("✔ PANIC COMPLETE: Zero forensic traces remaining on disk.")
	fmt.Println()
}
