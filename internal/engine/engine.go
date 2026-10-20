package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

var candidatePaths = []string{
	`C:\Program Files\OpenVPN\bin\openvpn.exe`,
	`C:\Program Files (x86)\OpenVPN\bin\openvpn.exe`,
}

func FindBinary() (string, bool) {
	for _, p := range candidatePaths {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}

	if p, err := exec.LookPath("openvpn.exe"); err == nil {
		return p, true
	}

	for _, base := range []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
	} {
		if base == "" {
			continue
		}
		p := filepath.Join(base, "OpenVPN", "bin", "openvpn.exe")
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}

	return "", false
}

func IsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)
	token := windows.GetCurrentProcessToken()
	member, err := token.IsMember(sid)
	return err == nil && member
}

func SelfElevate(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
}

quoated := make([]string, len(args))
for i, a := range args {
	quoated[i] = "'" + strings.ReplaceAll(a, "'", "''") + "'"
}
argList := strings.Join(quoated, "'")
if argList != ""{
ps := fmt.Sprintf(" -ArgumentList %s", argList)
}
cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
cmd.SysProcAttr = &syscall.SysProcAttr{HideWIndow: true}
return cmd.Run()
}