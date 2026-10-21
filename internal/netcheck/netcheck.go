package netcheck

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

type Result struct {
	ID       string
	Hostname string
	Latency  time.Duration
	Err      error
}

func TCPPing(ctx context.Context, host string, port int, timeout time.Duration) (time.Duration, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	start := time.Now()
	var d net.Dialer
	d.Timeout = timeout
	conn, err := d.DialContext(ctx, "tcp4", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}

func PingAll(ctx context.Context, hosts []string) []Result {
	results := make([]Result, len(hosts))
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Add(1)
		go func(i int, h string) {
			defer wg.Done()
			lat, err := TCPPing(ctx, h, 443, 4*time.Second)
			results[i] = Result{
				ID:       h,
				Hostname: h,
				Latency:  lat,
				Err:      err,
			}
		}(i, h)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		if results[i].Err != nil {
			return false
		}
		if results[j].Err != nil {
			return true
		}
		return results[i].Latency < results[j].Latency
	})

	return results
}
