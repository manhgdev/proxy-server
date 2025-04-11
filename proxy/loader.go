package proxy

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LoadProxies loads proxies from a file
func LoadProxies(filename string, pm *ProxyManager) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening proxy file: %v", err)
	}
	defer file.Close()

	var proxies []*Proxy
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse proxy URL based on format
		var proxy *Proxy

		// Check if proxy URL contains @ (user:pass@host:port format)
		if strings.Contains(line, "@") {
			parts := strings.Split(line, "@")
			if len(parts) == 2 {
				auth := strings.Split(parts[0], ":")
				if len(auth) == 2 {
					proxy = &Proxy{
						URL:      fmt.Sprintf("http://%s", parts[1]),
						Username: auth[0],
						Password: auth[1],
						LastUsed: time.Time{},
					}
				}
			}
		} else if strings.Count(line, ":") == 3 {
			// ip:port:user:pass format
			parts := strings.Split(line, ":")
			if len(parts) == 4 {
				proxy = &Proxy{
					URL:      fmt.Sprintf("http://%s:%s", parts[0], parts[1]),
					Username: parts[2],
					Password: parts[3],
					LastUsed: time.Time{},
				}
			}
		} else {
			// ip:port or domain:port format
			proxy = &Proxy{
				URL:      fmt.Sprintf("http://%s", line),
				LastUsed: time.Time{},
			}
		}

		if proxy != nil && proxy.URL != "" {
			proxies = append(proxies, proxy)
		}
	}

	pm.mu.Lock()
	pm.proxies = proxies
	pm.used = make(map[string]time.Time)
	pm.mu.Unlock()

	log.Printf("[INFO] Loaded %d proxies from %s", len(proxies), filename)
	return scanner.Err()
}

// MonitorProxyList monitors a proxy list file for changes
func MonitorProxyList(filename string, pm *ProxyManager) {
	var lastMod time.Time

	// Get initial file modification time without loading immediately
	stat, err := os.Stat(filename)
	if err == nil {
		lastMod = stat.ModTime()
	}

	for {
		time.Sleep(5 * time.Second)

		stat, err := os.Stat(filename)
		if err == nil {
			if stat.ModTime() != lastMod {
				if err := LoadProxies(filename, pm); err != nil {
					log.Printf("[ERROR] Error reloading proxies: %v", err)
				}
				lastMod = stat.ModTime()
			}
		}
	}
}
