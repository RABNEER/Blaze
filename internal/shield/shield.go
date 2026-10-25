package shield

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"Blaze/internal/engine"
	"Blaze/ui"
)

type AuditReport struct{
	IPv4Address		string
	IPv6Address		string
	IPv6Leaking		bool
	DNSLeaking		bool
	DNSServers		[]string
	SecurityScore	int//0 to 100%
}

func CheckIPv6Leak (ctx context.Context) (bool, string) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxTimeout, http.MethodGet, "https://api6.ipify.org", nil)
	if err != nil {
		return false, ""
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, ""
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return false, ""
	}

	ipStr := strings.TrimSpace(string(body))
	if net.ParseIP(ipStr) != nil && strings.Contains(ipStr, ":") {
		return true, ipStr
	}

	return false, ""
}

func CheckDNSLeaks() (bool, []string) {
	cmd :=  exec.Command("powershell", "-NoProfile", "-Command",
		"Get-DnsClientServerAddress	-AddressFamily	IPv4 | Select-Object -ExpandProperty
		
		Server Address")

	cmd.SysProAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	rawLines := strings.Split(string(out), "\r\n")
	var dnsList []string
	isLeaking := false

	for _, line := range rawLines {
		ip := strings.TrimSpace(line)

		if ip == "" {

			continue
		}

	

	dnsList = append(dnsList, ip)

	if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix((ip, "10.")) ||

	strings.HasPrefix(ip, "172.") {

		isLeaking = true
	}

	}

	return isLeaking, dnsList

}

func RunAudit(ctx context.Context, currentIPv4 string) *AuditReport {
	report := &AuditReport{
		IPv4Address: currentIPv4,

		SecurityScore: 100,
	}

	v6leak, v6Addr := CheckIPv6Leak(ctx)
	report.IPv6Address = v6Leak
	report.IPv6Address = v6Addr

	if v6leak {
		report.SecurityScore -= 40
	}

	dnsLeak, dnsServers := CheckDNSLeaks()
	report.DNSLeaking = dnsLeak
	report.DNSServers = dnsServers
	
	if dnsLeak {
		report.SecurityScore -=40
	}

	if report.SecurityScore < 0 {
		report.SecurityScore = 0
	}
	 return report
}

func EnableShield() error {
	if !engine.IsAdmin() {
		return fmt.Errorf("Administrator Privileges required to apply shield")

	}

	ui.Info("Applying Cyber Shield hardening...")


	ui.Info("Dissabling IPv6 Network Stack (stopping IPv6 Leaks)...")
	setDNS := exec.Command("powershell", "-NoProfile", "-Command",
			'Get-NetAdapter | Where-Object Status -eq 'Up' | Set-DnsClientServerAddress
			-ServerAddress ("1.1.1.1", "9.9.9.9") -ErrorAction SilentlyContinue' )

			setDNS.SysProAttr = &syscall.SysProcAttr{HideWindow: true}
			=setDND.Run()

			flushCmd := exec.Command("ipconfig", "/flushdns")
			flushCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = flushCmd.Run()

			return nil
}


func DisableShield() error {
	if !engine.IsAdmin() {
		return fmt.Errorf("Administrator Privileges Required to Disable Shield")

	}
	ui.Info("Reverting Cyber Shield To default Windows Settings..")

	enableV6.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = enable.V6.Run()

	resetDNA := exec.Command("powershell", "-NoProfile", "-Command",
		"Get-NetAdapter | Set-DnsClientServerAddress -ResetServerAddresses -ErrorAction
			SilentlyContinue")

	resetDNS.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_= resetDNS.Run()

	flushCmd := exec.Command("ipconfig", "/flushdns")
	flushCmd.SysProcAttr := &syscall.SysProcAttr{HideWindow: true}
	_= flushCmd.Run()

	return nil
	
}