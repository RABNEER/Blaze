package spoof

import (
	"crypto/rand"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"Blaze/internal/engine"
	"Blaze/ui"
)

// GenerateRandomMAC creates a valid, locally-administered unicast MAC address.
// The second nibble must be 2, 6, A, or E to satisfy IEEE 802 standards.
func GenerateRandomMAC() string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)

	// Enforce locally administered unicast: set bit 1 to 1, bit 0 to 0
	validSecondNibbles := []byte{'2', '6', 'A', 'E'}
	secondNibble := validSecondNibbles[buf[0]%4]

	firstByteHex := fmt.Sprintf("%1X%c", buf[0]>>4, secondNibble)

	return fmt.Sprintf("%s-%02X-%02X-%02X-%02X-%02X",
		firstByteHex, buf[1], buf[2], buf[3], buf[4], buf[5])
}

// GetActiveAdapter finds the name of the primary active network adapter (Wi-Fi or Ethernet).
func GetActiveAdapter() (string, string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`$a = Get-NetAdapter | Where-Object Status -eq 'Up' | Select-Object -First 1; if ($a) { "$($a.Name)|$($a.MacAddress)" }`)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to query network adapters: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return "", "", fmt.Errorf("no active network adapter found")
	}

	parts := strings.Split(trimmed, "|")
	if len(parts) < 2 {
		return parts[0], "unknown", nil
	}
	return parts[0], parts[1], nil
}

// ApplySpoof sets a new random MAC address on the active adapter and cycles it.
func ApplySpoof() error {
	if !engine.IsAdmin() {
		return fmt.Errorf("administrator privileges required to spoof MAC address")
	}

	adapterName, oldMAC, err := GetActiveAdapter()
	if err != nil {
		return err
	}

	newMAC := GenerateRandomMAC()

	ui.Info(fmt.Sprintf("Primary Adapter: %s", adapterName))
	ui.Info(fmt.Sprintf("Current Hardware MAC: %s", oldMAC))
	ui.Info(fmt.Sprintf("Injecting Spoofed MAC: %s...", newMAC))

	psScript := fmt.Sprintf(
		`Set-NetAdapter -Name '%s' -MacAddress '%s' -Confirm:$false; Restart-NetAdapter -Name '%s'`,
		adapterName, newMAC, adapterName,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply MAC address (adapter driver may require manual registry override): %w", err)
	}

	ui.Success(fmt.Sprintf("MAC Address spoofed successfully to %s!", newMAC))
	ui.Info("Adapter cycled. Hardware identity camouflaged on local Wi-Fi.")
	return nil
}

// ResetSpoof clears any spoofed MAC address and restores the factory hardware address.
func ResetSpoof() error {
	if !engine.IsAdmin() {
		return fmt.Errorf("administrator privileges required to reset MAC address")
	}

	adapterName, currentMAC, err := GetActiveAdapter()
	if err != nil {
		return err
	}

	ui.Info(fmt.Sprintf("Resetting MAC address for %s (Current: %s)...", adapterName, currentMAC))

	psScript := fmt.Sprintf(
		`Set-NetAdapter -Name '%s' -MacAddress '' -Confirm:$false; Restart-NetAdapter -Name '%s'`,
		adapterName, adapterName,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset MAC address: %w", err)
	}

	ui.Success("Hardware MAC address restored to factory default.")
	return nil
}
