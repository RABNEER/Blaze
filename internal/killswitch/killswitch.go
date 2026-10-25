package killswitch

import (
	"fmt"
	"os/exec"
	"syscall"

	"Blaze/internal/engine"
	"Blaze/ui"
)

const (
	RuleBlockAll   = "Blaze-KS-BlockAll"
	RuleAllowVPN   = "Blaze-KS-AllowVPN"
	RuleAllowDHCP  = "Blaze-KS-AllowDHCP"
	RuleAllowLocal = "Blaze-KS-AllowLocal"
)

// Enable enforces Windows Firewall rules that completely isolate outbound traffic,
// allowing ONLY OpenVPN traffic to exit. If OpenVPN crashes, all internet is blocked.
func Enable() error {
	if !engine.IsAdmin() {
		return fmt.Errorf("administrator privileges required to activate Kill Switch")
	}

	ovpnBin, found := engine.FindBinary()
	if !found {
		return fmt.Errorf("openvpn.exe not found")
	}

	ui.Info("Activating Hardware Kill-Switch...")

	// 1. Clean any old kill switch rules first
	_ = Disable()

	// 2. Rule: Allow DHCP (keeps your router IP lease active)
	dhcpCmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%s", RuleAllowDHCP),
		"dir=out", "action=allow", "protocol=UDP", "localport=68", "remoteport=67",
	)
	dhcpCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = dhcpCmd.Run()

	// 3. Rule: Allow OpenVPN binary outbound
	vpnCmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%s", RuleAllowVPN),
		"dir=out", "action=allow", fmt.Sprintf("program=%s", ovpnBin),
	)
	vpnCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = vpnCmd.Run()

	// 4. Rule: Block all other outbound traffic!
	blockCmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%s", RuleBlockAll),
		"dir=out", "action=block",
	)
	blockCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := blockCmd.Run(); err != nil {
		return fmt.Errorf("failed to apply firewall block rule: %w", err)
	}

	ui.Success("Kill-Switch ARMED: Outbound plaintext traffic strictly blocked.")
	ui.Info("If the tunnel drops, Windows will prevent all unencrypted leaks.")
	return nil
}

// Disable removes the firewall rules and restores full unblocked connectivity.
func Disable() error {
	rules := []string{RuleBlockAll, RuleAllowVPN, RuleAllowDHCP, RuleAllowLocal}
	for _, r := range rules {
		delCmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule",
			fmt.Sprintf("name=%s", r),
		)
		delCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = delCmd.Run()
	}
	return nil
}
