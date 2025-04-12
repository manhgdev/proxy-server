package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// TestEndpoint đại diện cho một endpoint để kiểm tra
type TestEndpoint struct {
	Name     string
	URL      string
	Protocol string
}

// ProxyTest lưu trữ kết quả kiểm tra
type ProxyTest struct {
	Endpoint     TestEndpoint
	StatusCode   int
	ResponseBody string
	Error        string
	Duration     time.Duration
	ProxyType    string // Loại proxy đã sử dụng
}

// ProxyConfig cấu hình cho một proxy để test
type ProxyConfig struct {
	URL      string
	Type     string
	Disabled bool
}

// Cấu hình
var (
	httpProxyURL   = "http://localhost:8081"
	socks5ProxyURL = "socks5://localhost:8081"
	testTimeout    = 30 * time.Second
	showFullBody   = false
	endpoints      = []TestEndpoint{
		{Name: "ZM API", URL: "https://api.zm.io.vn/check-ip", Protocol: "HTTPS"},
		{Name: "IP4.me", URL: "http://ip4.me/api/", Protocol: "HTTP"},
		{Name: "IP-API", URL: "http://ip-api.com/json", Protocol: "HTTP"},
		{Name: "IPIFY", URL: "https://api.ipify.org?format=json", Protocol: "HTTPS"},
		{Name: "HTTPBin", URL: "https://httpbin.org/ip", Protocol: "HTTPS"},
	}
	proxyConfigs = []ProxyConfig{
		{URL: httpProxyURL, Type: "HTTP", Disabled: false},
		{URL: socks5ProxyURL, Type: "SOCKS5", Disabled: false},
	}
)

// Hàm in màu cho terminal
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

func main() {
	fmt.Printf("%s=== PROXY SERVER TEST ===%s\n", colorCyan, colorReset)
	fmt.Println()

	// Xử lý tham số dòng lệnh
	if len(os.Args) > 1 {
		for _, arg := range os.Args {
			if arg == "--full" || arg == "-f" {
				showFullBody = true
			} else if strings.HasPrefix(arg, "--http=") {
				httpProxyURL = strings.TrimPrefix(arg, "--http=")
				updateProxyConfig("HTTP", httpProxyURL, false)
			} else if strings.HasPrefix(arg, "--socks5=") {
				socks5ProxyURL = strings.TrimPrefix(arg, "--socks5=")
				updateProxyConfig("SOCKS5", socks5ProxyURL, false)
			} else if arg == "--no-http" {
				updateProxyConfig("HTTP", "", true)
			} else if arg == "--no-socks5" {
				updateProxyConfig("SOCKS5", "", true)
			} else if strings.HasPrefix(arg, "--proxy=") {
				// Tương thích với phiên bản cũ
				proxyURL := strings.TrimPrefix(arg, "--proxy=")
				if strings.HasPrefix(proxyURL, "socks5://") {
					updateProxyConfig("SOCKS5", proxyURL, false)
					updateProxyConfig("HTTP", "", true) // Disable HTTP test
				} else {
					updateProxyConfig("HTTP", proxyURL, false)
					updateProxyConfig("SOCKS5", "", true) // Disable SOCKS5 test
				}
			}
		}
	}

	// Hiển thị các proxy đang test
	for _, cfg := range proxyConfigs {
		if !cfg.Disabled {
			fmt.Printf("Proxy (%s): %s%s%s\n", cfg.Type, colorYellow, cfg.URL, colorReset)
		}
	}
	fmt.Printf("Timeout: %s%s%s\n", colorYellow, testTimeout, colorReset)
	fmt.Println()

	allResults := make(map[string][]ProxyTest)
	var wg sync.WaitGroup
	var allMutex sync.Mutex

	// Chạy test cho từng loại proxy
	for _, proxyCfg := range proxyConfigs {
		if proxyCfg.Disabled {
			continue
		}

		wg.Add(1)
		go func(cfg ProxyConfig) {
			defer wg.Done()

			// Parse proxy URL
			parsedProxyURL, err := url.Parse(cfg.URL)
			if err != nil {
				fmt.Printf("%sLỗi parse proxy URL %s: %v%s\n", colorRed, cfg.URL, err, colorReset)
				return
			}

			// Cấu hình đặc biệt cho SOCKS5
			skipVerify := cfg.Type == "SOCKS5"

			// Tạo transport với proxy
			transport := &http.Transport{
				Proxy: http.ProxyURL(parsedProxyURL),
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: skipVerify, // Bỏ qua xác thực SSL khi dùng SOCKS5
				},
			}

			// Tạo client với transport
			client := &http.Client{
				Transport: transport,
				Timeout:   testTimeout,
			}

			results := runTests(client, endpoints, cfg.Type)

			allMutex.Lock()
			allResults[cfg.Type] = results
			allMutex.Unlock()
		}(proxyCfg)
	}

	wg.Wait()

	// Hiển thị kết quả tổng hợp
	fmt.Printf("\n%s=== TEST SUMMARY ===%s\n", colorBold+colorCyan, colorReset)

	// Hiển thị kết quả cho từng loại proxy
	for proxyType, results := range allResults {
		successCount := 0
		for _, r := range results {
			if r.Error == "" && r.StatusCode >= 200 && r.StatusCode < 300 {
				successCount++
			}
		}

		// Màu sắc trạng thái
		statusColor := colorYellow
		if successCount == len(results) {
			statusColor = colorGreen
		} else if successCount == 0 {
			statusColor = colorRed
		}

		fmt.Printf("%s%s Proxy:%s Thành công %s%d/%d%s\n",
			colorBold, proxyType, colorReset,
			statusColor, successCount, len(results), colorReset)
	}
}

// Cập nhật cấu hình proxy trong mảng toàn cục
func updateProxyConfig(proxyType, proxyURL string, disabled bool) {
	for i, cfg := range proxyConfigs {
		if cfg.Type == proxyType {
			if proxyURL != "" {
				proxyConfigs[i].URL = proxyURL
			}
			proxyConfigs[i].Disabled = disabled
			break
		}
	}
}

// Chạy test tới các endpoint sử dụng client đã được cấu hình
func runTests(client *http.Client, endpoints []TestEndpoint, proxyType string) []ProxyTest {
	results := make([]ProxyTest, 0)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	fmt.Printf("%s=== TESTING %s PROXY ===%s\n", colorBold+colorCyan, proxyType, colorReset)

	// Kiểm tra các endpoint song song
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep TestEndpoint) {
			defer wg.Done()
			result := testEndpoint(client, ep, proxyType)

			mutex.Lock()
			results = append(results, result)
			displayResult(result)
			mutex.Unlock()
		}(endpoint)
	}

	wg.Wait()
	return results
}

func testEndpoint(client *http.Client, endpoint TestEndpoint, proxyType string) ProxyTest {
	result := ProxyTest{
		Endpoint:  endpoint,
		ProxyType: proxyType,
	}

	// Đo thời gian
	startTime := time.Now()

	// Tạo request
	req, err := http.NewRequest("GET", endpoint.URL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("Lỗi tạo request: %v", err)
		return result
	}

	// Thêm User-Agent
	req.Header.Set("User-Agent", "ProxyTester/1.0")

	// Gửi request
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("Lỗi gửi request: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(startTime)

	// Đọc response (giới hạn 10KB để tránh quá lớn)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10240))
	if err != nil {
		result.Error = fmt.Sprintf("Lỗi đọc response: %v", err)
		return result
	}

	result.ResponseBody = string(bodyBytes)
	return result
}

func displayResult(result ProxyTest) {
	fmt.Printf("%s● %s %s (%s):%s\n",
		colorBold,
		result.ProxyType,
		result.Endpoint.Name,
		result.Endpoint.Protocol,
		colorReset)

	if result.Error != "" {
		fmt.Printf("  %sLỗi: %s%s\n", colorRed, result.Error, colorReset)
		return
	}

	// Hiển thị status
	statusColor := colorGreen
	if result.StatusCode >= 400 {
		statusColor = colorRed
	} else if result.StatusCode >= 300 {
		statusColor = colorYellow
	}

	fmt.Printf("  Status: %s%d%s\n", statusColor, result.StatusCode, colorReset)
	fmt.Printf("  Time: %s%.2fms%s\n", colorBlue, float64(result.Duration)/float64(time.Millisecond), colorReset)

	// Hiển thị IP từ response nếu có thể
	ip := extractIP(result.ResponseBody)
	if ip != "" {
		fmt.Printf("  IP: %s%s%s\n", colorPurple, ip, colorReset)
	}

	// Hiển thị body nếu cần
	if showFullBody {
		fmt.Printf("  Body: %s\n", formatJSON(result.ResponseBody))
	}

	fmt.Println()
}

// Trích xuất IP từ các định dạng response phổ biến
func extractIP(body string) string {
	// Thử parse JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		// Kiểm tra các trường IP thông dụng
		for _, field := range []string{"ip", "IP", "query", "origin"} {
			if ip, ok := data[field].(string); ok {
				return ip
			}
		}
	}

	// Thử với định dạng IP4.me: "IPv4,149.19.197.103,v1.1,,,See http://ip6.me/docs/ for api documentation"
	if strings.HasPrefix(body, "IPv4,") || strings.HasPrefix(body, "IPv6,") {
		parts := strings.Split(body, ",")
		if len(parts) >= 2 && isValidIP(parts[1]) {
			return parts[1]
		}
	}

	// Thử với định dạng IP4.me (plain text) - phiên bản cũ
	if strings.Count(body, ".") == 3 && len(body) < 20 {
		return strings.TrimSpace(body)
	}

	return ""
}

// Kiểm tra xem một chuỗi có phải là IP hợp lệ không
func isValidIP(ip string) bool {
	// Đơn giản nhưng hiệu quả cho hầu hết trường hợp
	ip = strings.TrimSpace(ip)
	return strings.Count(ip, ".") == 3 && len(ip) >= 7 && len(ip) <= 15
}

// Format JSON để dễ đọc
func formatJSON(jsonStr string) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(jsonStr), "    ", "  "); err == nil {
		return prettyJSON.String()
	}
	return jsonStr
}
