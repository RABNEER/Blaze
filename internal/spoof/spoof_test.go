package spoof

import (
	"regexp"
	"testing"
)

func TestGenerateRandomMAC(t *testing.T) {
	re := regexp.MustCompile(`^[0-9A-F][26AE]-[0-9A-F]{2}-[0-9A-F]{2}-[0-9A-F]{2}-[0-9A-F]{2}-[0-9A-F]{2}$`)

	for i := 0; i < 20; i++ {
		mac := GenerateRandomMAC()
		if !re.MatchString(mac) {
			t.Errorf("GenerateRandomMAC() produced invalid MAC %q", mac)
		}
	}
}
