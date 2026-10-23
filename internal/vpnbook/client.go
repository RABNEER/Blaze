package vpnbook

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	BaseURL     = "https://www.vpnbook.com"
	OpenVPNPage = BaseURL + "/freevpn/openvpn"
	ConfigAPI   = BaseURL + "/api/openvpn"
	httpTimeout = 15 * time.Second
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"
)

type Credentials struct {
	Username    string
	Password    string
	LastUpdated string
	FetchedAt   time.Time
}

var (
	reCredBlocks = regexp.MustCompile(`class="font-mono text-lg font-semibold text-gray-900">([^<]{3,20})</code>`)
	reLabeled    = regexp.MustCompile(`(?s)Username.*?<code[^>]*>([^<]+)</code>.*?Password.*?<code[^>]*>([^<]+)</code>`)
	reUpdated    = regexp.MustCompile(`Last updated:\s*([A-Za-z]+ \d{1,2}, \d{4})`)
)

func fetchPage(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vpnbook returned HTTP %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func GetCredentials(ctx context.Context) (*Credentials, error) {
	html, err := fetchPage(ctx, OpenVPNPage)
	if err != nil {
		return nil, fmt.Errorf("cannot reach vpnbook.com: %w", err)
	}

	creds := &Credentials{Username: "vpnbook"}

	var password string
	if m := reLabeled.FindStringSubmatch(html); m != nil {
		creds.Username = strings.TrimSpace(m[1])
		password = strings.TrimSpace(m[2])
	} else if blocks := reCredBlocks.FindAllStringSubmatch(html, -1); len(blocks) >= 2 {
		creds.Username = strings.TrimSpace(blocks[0][1])
		password = strings.TrimSpace(blocks[1][1])
	}

	if creds.Username == "" || password == "" {
		return nil, fmt.Errorf("could not extract credentials from the page (layout may have changed)")
	}
	if m := reUpdated.FindStringSubmatch(html); m != nil {
		creds.LastUpdated = strings.TrimSpace(m[1])
	}
	creds.Password = password
	creds.FetchedAt = time.Now()
	return creds, nil
}

func DownloadConfig(ctx context.Context, hostname, protocol string) ([]byte, error) {
	url := fmt.Sprintf("%s?hostname=%s&protocol=%s", ConfigAPI, hostname, protocol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config API returned HTTP %d (%s)", resp.StatusCode, protocol)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if !strings.Contains(string(body), "client") {
		return nil, fmt.Errorf("config API returned unexpected content (missing 'client' directive)")
	}
	return body, nil
}
