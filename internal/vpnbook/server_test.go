package vpnbook

import (
	"testing"
)

func TestValidateProtocol(t *testing.T) {
	valid := []string{"tcp443", "tcp80", "udp53", "udp25000"}
	for _, p := range valid {
		if !ValidateProtocol(p) {
			t.Errorf("ValidateProtocol(%q) = false; want true", p)
		}
	}

	invalid := []string{"tcp22", "http", "udp1194", "", "wireguard"}
	for _, p := range invalid {
		if ValidateProtocol(p) {
			t.Errorf("ValidateProtocol(%q) = true; want false", p)
		}
	}
}

func TestFindServer(t *testing.T) {
	servers := []Server{
		{ID: "us16", Name: "US Server 1", Hostname: "us16.vpnbook.com", Country: "US"},
		{ID: "ca149", Name: "Canada Server 1", Hostname: "ca149.vpnbook.com", Country: "CA"},
	}

	s := FindServer(servers, "us16")
	if s == nil || s.ID != "us16" {
		t.Fatalf("FindServer by ID failed: got %v", s)
	}

	s = FindServer(servers, "ca149.vpnbook.com")
	if s == nil || s.ID != "ca149" {
		t.Fatalf("FindServer by Hostname failed: got %v", s)
	}

	s = FindServer(servers, "nonexistent")
	if s != nil {
		t.Fatalf("FindServer expected nil; got %v", s)
	}
}
