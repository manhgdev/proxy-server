package proxy

import (
	"fmt"
	"strings"
	"time"
)

// ParseProxy parses a proxy string into a Proxy struct
func ParseProxy(line string) (*Proxy, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	// Check if proxy URL contains @ (user:pass@host:port format)
	if strings.Contains(line, "@") {
		parts := strings.Split(line, "@")
		if len(parts) == 2 {
			auth := strings.Split(parts[0], ":")
			if len(auth) == 2 {
				return &Proxy{
					URL:      fmt.Sprintf("http://%s", parts[1]),
					Username: auth[0],
					Password: auth[1],
					LastUsed: time.Time{},
				}, nil
			}
		}
	} else if strings.Count(line, ":") == 3 {
		// ip:port:user:pass format
		parts := strings.Split(line, ":")
		if len(parts) == 4 {
			return &Proxy{
				URL:      fmt.Sprintf("http://%s:%s", parts[0], parts[1]),
				Username: parts[2],
				Password: parts[3],
				LastUsed: time.Time{},
			}, nil
		}
	} else {
		// ip:port or domain:port format
		return &Proxy{
			URL:      fmt.Sprintf("http://%s", line),
			LastUsed: time.Time{},
		}, nil
	}

	return nil, fmt.Errorf("invalid proxy format")
}
