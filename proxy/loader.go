package proxy

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// ProxyType định nghĩa loại proxy
type ProxyType string

const (
	ProxyTypeHTTP    ProxyType = "http"
	ProxyTypeSOCKS5  ProxyType = "socks5"
	ProxyTypeUnknown ProxyType = "unknown"
)

// LoadProxiesFromMultipleFiles tải proxy từ nhiều file
func LoadProxiesFromMultipleFiles(httpFile, socks5File string, pm *ProxyManager) error {
	// Tải các proxy HTTP
	if httpFile != "" {
		if err := LoadProxiesWithType(httpFile, ProxyTypeHTTP, pm); err != nil {
			log.Printf("[WARN] Error loading HTTP proxies: %v", err)
		}
	}

	// Tải các proxy SOCKS5
	if socks5File != "" {
		if err := LoadProxiesWithType(socks5File, ProxyTypeSOCKS5, pm); err != nil {
			log.Printf("[WARN] Error loading SOCKS5 proxies: %v", err)
		}
	}

	// Kiểm tra xem có proxy nào được tải không
	if pm.GetProxyCount() == 0 {
		return fmt.Errorf("no valid proxies found in any file")
	}

	return nil
}

// LoadProxies tải proxy từ một file duy nhất (cho tương thích ngược)
func LoadProxies(filename string, pm *ProxyManager) error {
	return LoadProxiesWithType(filename, ProxyTypeHTTP, pm)
}

// LoadProxiesWithType tải proxy từ file với type xác định
func LoadProxiesWithType(filename string, proxyType ProxyType, pm *ProxyManager) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening proxy file %s: %v", filename, err)
	}
	defer file.Close()

	var proxies []*Proxy
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Bỏ qua dòng trống và comment
		}

		// Parse proxy URL dựa vào định dạng
		proxy, err := ParseProxy(line)
		if err != nil {
			log.Printf("[WARN] Invalid proxy format in %s: %s, error: %v", filename, line, err)
			continue
		}

		// Gán type cho proxy
		proxy.Type = proxyType
		proxy.IsWorking = true // Giả định hoạt động ban đầu

		// Đảm bảo URL có prefix đúng với loại proxy
		if !strings.Contains(proxy.URL, "://") {
			if proxyType == ProxyTypeSOCKS5 {
				proxy.URL = "socks5://" + proxy.URL
			} else {
				proxy.URL = "http://" + proxy.URL
			}
		}

		proxies = append(proxies, proxy)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading proxy file %s: %v", filename, err)
	}

	pm.mu.Lock()
	for _, proxy := range proxies {
		pm.AddProxy(proxy)
	}
	pm.mu.Unlock()

	log.Printf("[INFO] Loaded %d %s proxies from %s", len(proxies), proxyType, filename)
	return nil
}

// MonitorProxyList giám sát các file proxy để cập nhật
func MonitorProxyList(httpFile, socks5File string, pm *ProxyManager) {
	var lastModHTTP, lastModSOCKS5 time.Time

	// Lấy thời gian sửa đổi ban đầu
	if httpFile != "" {
		if stat, err := os.Stat(httpFile); err == nil {
			lastModHTTP = stat.ModTime()
		}
	}

	if socks5File != "" {
		if stat, err := os.Stat(socks5File); err == nil {
			lastModSOCKS5 = stat.ModTime()
		}
	}

	for {
		time.Sleep(5 * time.Second)

		// Kiểm tra file HTTP proxy
		if httpFile != "" {
			if stat, err := os.Stat(httpFile); err == nil {
				if stat.ModTime() != lastModHTTP {
					log.Printf("[INFO] HTTP proxy file %s changed, reloading", httpFile)
					if err := LoadProxiesWithType(httpFile, ProxyTypeHTTP, pm); err != nil {
						log.Printf("[ERROR] Error reloading HTTP proxies: %v", err)
					}
					lastModHTTP = stat.ModTime()
				}
			}
		}

		// Kiểm tra file SOCKS5 proxy
		if socks5File != "" {
			if stat, err := os.Stat(socks5File); err == nil {
				if stat.ModTime() != lastModSOCKS5 {
					log.Printf("[INFO] SOCKS5 proxy file %s changed, reloading", socks5File)
					if err := LoadProxiesWithType(socks5File, ProxyTypeSOCKS5, pm); err != nil {
						log.Printf("[ERROR] Error reloading SOCKS5 proxies: %v", err)
					}
					lastModSOCKS5 = stat.ModTime()
				}
			}
		}
	}
}
