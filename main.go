package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"proxy/proxy"
)

func main() {
	// Create proxy manager
	pm := proxy.NewProxyManager()

	// Load proxies from file
	if err := proxy.LoadProxies("proxy_list.txt", pm); err != nil {
		log.Fatalf("[ERROR] Failed to load proxies: %v", err)
	}

	// Start monitoring proxy list
	go proxy.MonitorProxyList("proxy_list.txt", pm)

	// Start proxy server
	go func() {
		// Changed port from 8080 to 8081 for testing
		if err := proxy.StartProxyServer(pm, ":8081"); err != nil {
			log.Fatalf("[ERROR] Failed to start proxy server: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("[INFO] Shutting down server...")
}
