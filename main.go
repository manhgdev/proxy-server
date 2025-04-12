package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"proxy/proxy"
)

const (
	// Cấu hình file proxy
	httpProxyFile   = "proxy_http.txt"
	socks5ProxyFile = "proxy_sockets5.txt"

	// Cổng lắng nghe
	serverPort = ":8081"
)

func main() {
	log.Println("[INFO] Khởi động proxy server")

	// Tạo proxy manager
	pm := proxy.NewProxyManager()

	// Tải proxy từ nhiều file
	if err := proxy.LoadProxiesFromMultipleFiles(httpProxyFile, socks5ProxyFile, pm); err != nil {
		log.Fatalf("[ERROR] Failed to load proxies: %v", err)
	}

	// Bắt đầu giám sát danh sách proxy
	go proxy.MonitorProxyList(httpProxyFile, socks5ProxyFile, pm)

	// Khởi động proxy server
	go func() {
		if err := proxy.StartProxyServer(pm, serverPort); err != nil {
			log.Fatalf("[ERROR] Failed to start proxy server: %v", err)
		}
	}()

	// Xử lý tắt graceful
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("[INFO] Shutting down server...")
}
