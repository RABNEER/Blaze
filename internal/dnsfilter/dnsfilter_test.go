package dnsfilter

import (
	"testing"
)

func TestIsBlocked(t *testing.T) {
	s := NewServer("127.0.0.1:5353", "1.1.1.1:53")

	tests := []struct {
		domain   string
		expected bool
	}{
		{"doubleclick.net", true},
		{"ad.doubleclick.net", true},
		{"telemetry.microsoft.com", true},
		{"v10.events.data.microsoft.com", true},
		{"google.com", false},
		{"github.com", false},
		{"vpnbook.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := s.IsBlocked(tt.domain)
			if got != tt.expected {
				t.Errorf("IsBlocked(%q) = %v; want %v", tt.domain, got, tt.expected)
			}
		})
	}
}

func TestBuildSinkholeResponse(t *testing.T) {
	// Sample raw DNS query for google.com
	rawQuery := []byte{
		0x12, 0x34, // ID
		0x01, 0x00, // Standard query
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0
		0x06, 'g', 'o', 'o', 'g', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,       // End of name
		0x00, 0x01, // Type A
		0x00, 0x01, // Class IN
	}

	domain, qEnd, err := ParseDomain(rawQuery)
	if err != nil {
		t.Fatalf("ParseDomain error: %v", err)
	}
	if domain != "google.com" {
		t.Fatalf("got domain %q; want %q", domain, "google.com")
	}

	resp := BuildSinkholeResponse(rawQuery, qEnd)
	if len(resp) < len(rawQuery)+16 {
		t.Fatalf("response too short: %d bytes", len(resp))
	}

	// Verify last 4 bytes are 0.0.0.0
	ipBytes := resp[len(resp)-4:]
	if ipBytes[0] != 0 || ipBytes[1] != 0 || ipBytes[2] != 0 || ipBytes[3] != 0 {
		t.Errorf("sinkhole IP = %v; want [0 0 0 0]", ipBytes)
	}
}
