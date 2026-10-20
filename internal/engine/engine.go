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

	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", "''") + "'"
	}

	argList := strings.Join(quoted, ",")
	ps := fmt.Sprintf("Start-Process -Filepath '%s' -Verb RunAs", exe)
	if argList != ""{
		ps += fmt.Sprintf(" -ArgumentList %s", argList)
	}

	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func ProcessAlive(pid int) bool{
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil{
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259
}

func KillProcess(pid int) error{
	cmd := exec.Command("taskkill", "/PID", fmt.Sprint(pid), "/F", "/T")
	return cmd.Run()
}

func TempAuthFile(username, password string) (string, func(), error) {
	f, err := os.CreateTemp("", "blaze-auth-*.txt")
	if err != nil {
		return "", nil, err
	}
	fmt.Fprintf(f, "%s\n%s\n", username, password)
	f.Close()
	cleanup := func() { _ = os.Remove(f.Name()) }
	return f.Name(), cleanup, nil
}

func ConfigCheckPath(serverID, proto string) string {
	return filepath.Join(cacheDir(), fmt.Sprintf("%s-%s.ovpn", serverID, proto))
}

func cacheDir() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		home, _ := os.UserHomeDir()
		local = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(local, "BlazeVPN", "configs")
}

