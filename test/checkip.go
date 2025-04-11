package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

func main() {
	// Cấu hình proxy
	proxyURL, err := url.Parse("http://127.0.0.1:8080")
	if err != nil {
		fmt.Printf("Lỗi parse proxy URL: %v\n", err)
		return
	}

	// Tạo transport với proxy
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// Tạo client với transport
	client := &http.Client{
		Transport: transport,
	}

	// Tạo request
	req, err := http.NewRequest("GET", "https://api.zm.io.vn/check-ip", nil)
	if err != nil {
		fmt.Printf("Lỗi tạo request: %v\n", err)
		return
	}

	// Gửi request
	fmt.Println("Đang gửi request qua proxy...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Lỗi gửi request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Đọc response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Lỗi đọc response: %v\n", err)
		return
	}

	// In kết quả
	// fmt.Printf("Status: %d\n", resp.StatusCode)
	// fmt.Printf("Headers: %v\n", resp.Header)
	fmt.Printf("Body: %s\n", string(body))
}
