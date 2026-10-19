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

var candidatePaths =[]string{
	'C:\Program Files\OpenVPN\bin\openvpn.exe'
	'C:\Program Files (x86)\OpenVPN\bin\openvpn.exe'
}

func FindBinary() (string, error) {
	for _, p := range candidatePaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if p, err := exec.LookPath("openvpn.exe"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("OpenVPN not found, Intall it with winget install OpenVPNTechnologies.OpenVPN")
}