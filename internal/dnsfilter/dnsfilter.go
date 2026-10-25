package dnsfilter

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"Blaze/ui"
)

// DefaultBlocklist contains high-frequency ad networks, telemetry, and tracking domains.
var DefaultBlocklist = []string{
	"doubleclick.net",
	"google-analytics.com",
	"googletagmanager.com",
	"telemetry.microsoft.com",
	"v10.events.data.microsoft.com",
	"watson.telemetry.microsoft.com",
	"adnxs.com",
	"ads.facebook.com",
	"graph.instagram.com",
	"ads.tiktok.com",
	"analytics.tiktok.com",
	"scorecardresearch.com",
	"outbrain.com",
	"taboola.com",
	"criteo.com",
	"quantserve.com",
	"rubiconproject.com",
	"advertising.com",
	"adcolony.com",
	"app-measurement.com",
	"appsflyer.com",
	"adjust.com",
	"branch.io",
	"hotjar.com",
	"mixpanel.com",
	"segment.io",
}

// Stats tracks live traffic and blocked packets.
type Stats struct {
	TotalQueries   uint64
	BlockedQueries uint64
	AllowedQueries uint64
}

// Server represents our local DNS Sinkhole instance.
type Server struct {
	ListenAddr   string
	UpstreamDNS  string
	Blocklist    map[string]bool
	Stats        Stats
	mu           sync.RWMutex
	conn         *net.UDPConn
}

// NewServer initializes the sinkhole with the blocklist and upstream resolver (Quad9/Cloudflare).
func NewServer(listenAddr, upstreamDNS string) *Server {
	if listenAddr == "" {
		listenAddr = "127.0.0.1:53"
	}
	if upstreamDNS == "" {
		upstreamDNS = "1.1.1.1:53"
	}

	bMap := make(map[string]bool)
	for _, domain := range DefaultBlocklist {
		bMap[strings.ToLower(domain)] = true
	}

	return &Server{
		ListenAddr:  listenAddr,
		UpstreamDNS: upstreamDNS,
		Blocklist:   bMap,
	}
}

// ParseDomain extracts the queried domain name from the raw DNS query packet (RFC 1035).
func ParseDomain(packet []byte) (string, int, error) {
	if len(packet) < 12 {
		return "", 0, fmt.Errorf("packet too short")
	}

	// Question section starts at byte 12
	idx := 12
	var parts []string

	for idx < len(packet) {
		length := int(packet[idx])
		if length == 0 {
			idx++
			break
		}
		idx++
		if idx+length > len(packet) {
			return "", 0, fmt.Errorf("malformed packet")
		}
		parts = append(parts, string(packet[idx:idx+length]))
		idx += length
	}

	domain := strings.ToLower(strings.Join(parts, "."))
	return domain, idx + 4, nil // idx + 4 skips QTYPE and QCLASS
}

// IsBlocked checks if the queried domain or any of its parent domains are in the blocklist.
func (s *Server) IsBlocked(domain string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Blocklist[domain] {
		return true
	}

	// Subdomain check (e.g. ad.service.doubleclick.net matches doubleclick.net)
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts)-1; i++ {
		parent := strings.Join(parts[i:], ".")
		if s.Blocklist[parent] {
			return true
		}
	}
	return false
}

// BuildSinkholeResponse constructs an RFC-1035 DNS response returning 0.0.0.0 for blocked domains.
func BuildSinkholeResponse(req []byte, qEnd int) []byte {
	resp := make([]byte, 0, len(req)+16)

	// Copy original request header + question
	resp = append(resp, req[:qEnd]...)

	// Modify Header: Set QR bit = 1 (Response), RA = 1, RCODE = 0 (NoError)
	resp[2] = 0x81
	resp[3] = 0x80

	// Set Answer Count = 1 (Bytes 6 & 7)
	resp[6] = 0x00
	resp[7] = 0x01

	// Answer Section:
	// Pointer to question name: 0xC00C (offset 12)
	resp = append(resp, 0xC0, 0x0C)
	// Type A (0x0001), Class IN (0x0001)
	resp = append(resp, 0x00, 0x01, 0x00, 0x01)
	// TTL: 60 seconds (0x0000003C)
	resp = append(resp, 0x00, 0x00, 0x00, 0x3C)
	// Data length: 4 bytes for IPv4 (0x0004)
	resp = append(resp, 0x00, 0x04)
	// IP address: 0.0.0.0 (Sinkholed!)
	resp = append(resp, 0x00, 0x00, 0x00, 0x00)

	return resp
}

// ForwardQuery sends safe DNS queries to the upstream encrypted DNS provider (1.1.1.1).
func (s *Server) ForwardQuery(req []byte) ([]byte, error) {
	conn, err := net.DialTimeout("udp", s.UpstreamDNS, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write(req); err != nil {
		return nil, err
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Start launches the UDP listener and begins filtering DNS requests in real-time.
func (s *Server) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", s.ListenAddr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP port 53 (ensure you have Administrator rights or another DNS service isn't occupying port 53): %w", err)
	}
	s.conn = conn
	defer conn.Close()

	ui.Success(fmt.Sprintf("DNS Sinkhole active on %s (Forwarding to %s)", s.ListenAddr, s.UpstreamDNS))
	ui.Info(fmt.Sprintf("Loaded %d tracking & ad domains into the active blocklist.", len(s.Blocklist)))
	ui.Info("Press Ctrl+C to terminate sinkhole.")
	fmt.Println()

	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		n, clientAddr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		atomic.AddUint64(&s.Stats.TotalQueries, 1)
		req := make([]byte, n)
		copy(req, buf[:n])

		go func(data []byte, target net.Addr) {
			domain, qEnd, err := ParseDomain(data)
			if err != nil {
				return
			}

			if s.IsBlocked(domain) {
				atomic.AddUint64(&s.Stats.BlockedQueries, 1)
				sinkholeResp := BuildSinkholeResponse(data, qEnd)
				_, _ = conn.WriteTo(sinkholeResp, target)
				ui.Error(fmt.Sprintf("🛡️ [BLOCKED] %s -> 0.0.0.0", domain))
			} else {
				atomic.AddUint64(&s.Stats.AllowedQueries, 1)
				resp, err := s.ForwardQuery(data)
				if err == nil {
					_, _ = conn.WriteTo(resp, target)
					ui.Success(fmt.Sprintf("✔ [ALLOWED] %s", domain))
				}
			}
		}(req, clientAddr)
	}
}
