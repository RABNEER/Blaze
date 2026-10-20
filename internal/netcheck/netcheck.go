package netcheck

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

type result struct {
	Server	 interface{ GetID() string}
	ID		 string
	Hostname time.Duration
	Err		 error
}

func TCPPing (ctx context.Context, host string, port int, timeout time.Duration) (time.Duration, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	start := time.Now()
	var d net.Dialer
	d.Timeout = timeout
	conn, err := d.DialContext(ctx, "tcp4",  addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}

