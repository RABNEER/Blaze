package vpnbook

import (
	"context"
	"regexp"
	"strings"
)

type Server struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	Country  string `json:"country"`
}

var Protocols = []string{"tcp443", "tcp80", "udp53", "udp25000"}

var reServerObj = regexp.MustCompile(
	`\{"id":"([a-z0-9]+)","name":"([^"]+)","hostname":"([a-z0-9.]+)","ipAddress":"([^"]*)","countryCode":"([A-Z]{2})`)

var fallbackServers = []Server{
	{"us16", "US Server 1", "us16.vpnbook.com", "", "US"},
	{"us178", "US Server 2", "us178.vpnbook.com", "", "US"},
	{"ca149", "Canada Server 1", "ca149.vpnbook.com", "", "CA"},
	{"ca196", "Canada Server 2", "ca196.vpnbook.com", "", "CA"},
	{"uk205", "UK Server 1", "uk205.vpnbook.com", "", "GB"},
	{"uk68", "UK Server 2", "uk68.vpnbook.com", "", "GB"},
	{"de20", "Germany Server 1", "de20.vpnbook.com", "", "DE"},
	{"de220", "Germany Server 2", "de220.vpnbook.com", "", "DE"},
	{"fr200", "France Server 1", "fr200.vpnbook.com", "", "FR"},
	{"fr2311", "France Server 2", "fr2311.vpnbook.com", "", "FR"},
}

func GetServers(ctx context.Context) ([]Server, error) {
	html, err := fetchPage(ctx, OpenVPNPage)
	if err != nil {
		return fallbackServers, nil
	}

	unescaped := strings.ReplaceAll(html, `\"`, `"`)
	matches := reServerObj.FindAllStringSubmatch(unescaped, -1)
	if len(matches) == 0 {
		return fallbackServers, nil
	}
	seen := map[string]bool{}
	servers := make([]Server, 0, len(matches))
	for _, m := range matches {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		servers = append(servers, Server{
			ID:       m[1],
			Name:     m[2],
			Hostname: m[3],
			IP:       m[4],
			Country:  m[5],
		})
	}
	return servers, nil
}

func FindServer(servers []Server, idOrHost string) *Server {
	q := strings.ToLower(idOrHost)
	for i := range servers {
		s := &servers[i]
		if s.ID == q || s.Hostname == q || strings.TrimSuffix(s.Hostname, ".vpnbook.com") == q {
			return s
		}
	}
	return nil
}

func ValidateProtocol(p string) bool {
	for _, v := range Protocols {
		if v == p {
			return true
		}
	}
	return false
}
