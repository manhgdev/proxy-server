package proxy

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Proxy struct {
	URL         string
	Username    string
	Password    string
	LastUsed    time.Time
	FailCount   int       // Track consecutive failures
	LastChecked time.Time // Last time the proxy was health checked
	IsWorking   bool      // Flag to indicate if proxy is working
	Type        ProxyType // Type of proxy (HTTP, SOCKS5)
}

type ProxyManager struct {
	proxies       []*Proxy
	mu            sync.RWMutex
	used          map[string]time.Time
	testURL       string        // URL used for testing proxies
	maxRetries    int           // Maximum number of retries with different proxies
	maxFails      int           // Maximum allowed consecutive failures
	checkInterval time.Duration // Interval for health checks
	rand          *rand.Rand    // Sử dụng rand riêng để tránh xung đột
}

func NewProxyManager() *ProxyManager {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &ProxyManager{
		proxies:       make([]*Proxy, 0),
		used:          make(map[string]time.Time),
		testURL:       "http://ip4.me/api", // Default test URL
		maxRetries:    3,
		maxFails:      5,
		checkInterval: 5 * time.Minute,
		rand:          r,
	}
}

// SetTestURL changes the URL used for testing proxies
func (pm *ProxyManager) SetTestURL(url string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.testURL = url
}

// SetMaxRetries sets the maximum number of retries with different proxies
func (pm *ProxyManager) SetMaxRetries(retries int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.maxRetries = retries
}

// SetMaxFails sets the maximum allowed consecutive failures
func (pm *ProxyManager) SetMaxFails(fails int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.maxFails = fails
}

// SetCheckInterval sets the interval for health checks
func (pm *ProxyManager) SetCheckInterval(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.checkInterval = duration
}

func (pm *ProxyManager) LoadProxies(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open proxy list file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		proxy, err := parseProxy(line)
		if err != nil {
			continue
		}
		proxy.IsWorking = true // Assume working by default
		pm.proxies = append(pm.proxies, proxy)
	}

	if len(pm.proxies) == 0 {
		return fmt.Errorf("no valid proxies found in file")
	}

	// Start background health checking
	go pm.startHealthChecks()

	return nil
}

// startHealthChecks runs periodic health checks on all proxies
func (pm *ProxyManager) startHealthChecks() {
	ticker := time.NewTicker(pm.checkInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		pm.checkAllProxies()
	}
}

// checkAllProxies tests all proxies in the pool
func (pm *ProxyManager) checkAllProxies() {
	pm.mu.Lock()
	proxies := make([]*Proxy, len(pm.proxies))
	copy(proxies, pm.proxies)
	pm.mu.Unlock()

	for _, proxy := range proxies {
		isWorking := pm.testProxy(proxy)

		pm.mu.Lock()
		proxy.LastChecked = time.Now()
		proxy.IsWorking = isWorking
		if !isWorking {
			proxy.FailCount++
		} else {
			proxy.FailCount = 0 // Reset fail count on success
		}
		pm.mu.Unlock()
	}

	// Remove proxies with too many failures
	pm.cleanupFailedProxies()
}

// testProxy checks if a proxy is working by making a test request
func (pm *ProxyManager) testProxy(proxy *Proxy) bool {
	var proxyURL *url.URL
	var err error

	// Parse proxy URL based on format
	if strings.Contains(proxy.URL, "http://") {
		// URL already has scheme
		proxyURL, err = url.Parse(proxy.URL)
	} else {
		// Add scheme
		proxyURL, err = url.Parse("http://" + proxy.URL)
	}

	if err != nil {
		return false
	}

	// Set proxy authentication if needed
	if proxy.Username != "" && proxy.Password != "" {
		proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
	}

	// Create HTTP client with proxy
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	// Try to fetch the test URL
	resp, err := client.Get(pm.testURL)
	if err != nil {
		logger.Info("Proxy test failed for %s: %v", proxy.URL, err)
		return false
	}
	defer resp.Body.Close()

	// Check if status code is successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Info("Proxy test failed for %s: status code %d", proxy.URL, resp.StatusCode)
		return false
	}

	logger.Info("Proxy test successful for %s", proxy.URL)
	return true
}

// cleanupFailedProxies removes proxies with too many failures
func (pm *ProxyManager) cleanupFailedProxies() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var workingProxies []*Proxy
	for _, proxy := range pm.proxies {
		if proxy.FailCount < pm.maxFails {
			workingProxies = append(workingProxies, proxy)
		} else {
			logger.Info("Removing failed proxy: %s (failed %d times)", proxy.URL, proxy.FailCount)
		}
	}

	pm.proxies = workingProxies
}

// MarkProxyFailed marks a proxy as failed and increments its failure count
func (pm *ProxyManager) MarkProxyFailed(failedProxy *Proxy) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, proxy := range pm.proxies {
		if proxy.URL == failedProxy.URL {
			proxy.IsWorking = false
			proxy.FailCount++
			logger.Info("Marked proxy as failed: %s (fail count: %d)", proxy.URL, proxy.FailCount)
			break
		}
	}
}

// GetNextWorkingProxy returns the next working proxy
func (pm *ProxyManager) GetNextWorkingProxy(excludeURL string) *Proxy {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil
	}

	// Find proxy that is working, hasn't been used or used longest time ago
	var selectedProxy *Proxy
	var oldestUsedTime time.Time
	first := true

	for _, proxy := range pm.proxies {
		// Skip the excluded proxy and non-working proxies
		if proxy.URL == excludeURL || !proxy.IsWorking {
			continue
		}

		lastUsed, exists := pm.used[proxy.URL]
		if !exists {
			// If proxy hasn't been used, select it immediately
			selectedProxy = proxy
			break
		}

		if first || lastUsed.Before(oldestUsedTime) {
			oldestUsedTime = lastUsed
			selectedProxy = proxy
			first = false
		}
	}

	// Update last used time
	if selectedProxy != nil {
		pm.used[selectedProxy.URL] = time.Now()
	}

	return selectedProxy
}

func parseProxy(line string) (*Proxy, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	// Format: user:pass@ip:port
	if strings.Contains(line, "@") {
		parts := strings.Split(line, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid proxy format")
		}

		auth := strings.Split(parts[0], ":")
		if len(auth) != 2 {
			return nil, fmt.Errorf("invalid auth format")
		}

		return &Proxy{
			URL:      fmt.Sprintf("http://%s", parts[1]),
			Username: auth[0],
			Password: auth[1],
			LastUsed: time.Time{},
		}, nil
	}

	// Format: ip:port:user:pass
	parts := strings.Split(line, ":")
	if len(parts) == 4 {
		return &Proxy{
			URL:      fmt.Sprintf("http://%s:%s", parts[0], parts[1]),
			Username: parts[2],
			Password: parts[3],
			LastUsed: time.Time{},
		}, nil
	}

	// Format: ip:port
	if len(parts) == 2 {
		return &Proxy{
			URL:      fmt.Sprintf("http://%s:%s", parts[0], parts[1]),
			Username: "",
			Password: "",
			LastUsed: time.Time{},
		}, nil
	}

	// Format: ip
	if len(parts) == 1 {
		return &Proxy{
			URL:      fmt.Sprintf("http://%s", parts[0]),
			Username: "",
			Password: "",
			LastUsed: time.Time{},
		}, nil
	}

	return nil, fmt.Errorf("invalid proxy format")
}

func splitProxyLine(line string) []string {
	var parts []string
	var current string
	var inQuotes bool

	for _, char := range line {
		switch char {
		case ':':
			if !inQuotes {
				parts = append(parts, current)
				current = ""
			} else {
				current += string(char)
			}
		case '"':
			inQuotes = !inQuotes
		default:
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// GetRandomProxy trả về một proxy ngẫu nhiên
func (pm *ProxyManager) GetRandomProxy() *Proxy {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	workingProxies := []*Proxy{}
	for _, proxy := range pm.proxies {
		if proxy.IsWorking {
			workingProxies = append(workingProxies, proxy)
		}
	}

	if len(workingProxies) == 0 {
		logger.Error("No working proxies available")
		return nil
	}

	// Chọn ngẫu nhiên một proxy
	randomIndex := pm.rand.Intn(len(workingProxies))
	selectedProxy := workingProxies[randomIndex]

	// Cập nhật thời gian sử dụng cuối cùng
	pm.used[selectedProxy.URL] = time.Now()

	logger.Info("Selected random proxy: %s", selectedProxy.URL)
	return selectedProxy
}

func (pm *ProxyManager) GetProxyCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.proxies)
}

// AddProxy thêm một proxy vào manager
func (pm *ProxyManager) AddProxy(proxy *Proxy) {
	// Nếu đã tồn tại proxy với URL này, cập nhật thay vì thêm mới
	for i, p := range pm.proxies {
		if p.URL == proxy.URL {
			pm.proxies[i] = proxy
			return
		}
	}

	// Thêm proxy mới
	pm.proxies = append(pm.proxies, proxy)
}

// ProxySelector định nghĩa hàm lọc proxy theo tiêu chí
type ProxySelector func(*Proxy) bool

// GetRandomProxyWithFilter trả về một proxy ngẫu nhiên phù hợp với bộ lọc
func (pm *ProxyManager) GetRandomProxyWithFilter(selector ProxySelector) *Proxy {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil
	}

	// Lọc proxy phù hợp
	var eligibleProxies []*Proxy
	for _, proxy := range pm.proxies {
		if proxy.IsWorking && selector(proxy) {
			eligibleProxies = append(eligibleProxies, proxy)
		}
	}

	if len(eligibleProxies) == 0 {
		return nil
	}

	// Chọn một proxy ngẫu nhiên
	randomIndex := pm.rand.Intn(len(eligibleProxies))
	selectedProxy := eligibleProxies[randomIndex]

	// Cập nhật thời gian sử dụng
	pm.used[selectedProxy.URL] = time.Now()

	logger.Info("Selected random proxy: %s", selectedProxy.URL)
	return selectedProxy
}

// GetNextWorkingProxyWithFilter trả về proxy tiếp theo phù hợp với bộ lọc
func (pm *ProxyManager) GetNextWorkingProxyWithFilter(excludeURL string, selector ProxySelector) *Proxy {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil
	}

	// Tìm proxy đang hoạt động, chưa được sử dụng hoặc đã lâu không sử dụng
	var selectedProxy *Proxy
	var oldestUsedTime time.Time
	first := true

	for _, proxy := range pm.proxies {
		// Bỏ qua proxy bị loại trừ, proxy không hoạt động và proxy không phù hợp với bộ lọc
		if proxy.URL == excludeURL || !proxy.IsWorking || !selector(proxy) {
			continue
		}

		lastUsed, exists := pm.used[proxy.URL]
		if !exists {
			// Nếu proxy chưa từng được sử dụng, chọn ngay lập tức
			selectedProxy = proxy
			break
		}

		if first || lastUsed.Before(oldestUsedTime) {
			oldestUsedTime = lastUsed
			selectedProxy = proxy
			first = false
		}
	}

	if selectedProxy != nil {
		// Cập nhật thời gian sử dụng
		pm.used[selectedProxy.URL] = time.Now()
		logger.Info("Selected next proxy: %s", selectedProxy.URL)
	}

	return selectedProxy
}

// MarkProxySuccess đánh dấu proxy thành công
func (pm *ProxyManager) MarkProxySuccess(successProxy *Proxy) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, proxy := range pm.proxies {
		if proxy.URL == successProxy.URL {
			proxy.IsWorking = true
			proxy.FailCount = 0
			logger.Info("Marked proxy as successful: %s", proxy.URL)
			break
		}
	}
}
