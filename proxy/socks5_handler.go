package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"
)

// Các hằng số SOCKS5
const (
	SOCKS5_VERSION          = 0x05
	SOCKS5_CMD_CONNECT      = 0x01
	SOCKS5_ADDR_TYPE_IPV4   = 0x01
	SOCKS5_ADDR_TYPE_DOMAIN = 0x03
	SOCKS5_ADDR_TYPE_IPV6   = 0x04
)

// SOCKS5Config cấu hình cho SOCKS5 proxy
type SOCKS5Config struct {
	// SkipVerify không có tác dụng với xác thực SSL ở client
	// vì SOCKS5 chỉ chuyển tiếp TCP (layer 4), trong khi SSL handshake
	// diễn ra trực tiếp giữa client và server đích (layer 7)
	// Giữ lại để tương thích ngược với API hiện tại
	SkipVerify bool
}

// SOCKS5DefaultConfig trả về cấu hình mặc định
func SOCKS5DefaultConfig() *SOCKS5Config {
	return &SOCKS5Config{
		SkipVerify: true, // Giá trị này không ảnh hưởng đến xác thực SSL của client
	}
}

var socks5Config = SOCKS5DefaultConfig()

// Cấu hình SOCKS5
func ConfigureSOCKS5(config *SOCKS5Config) {
	if config != nil {
		socks5Config = config
	}
}

// Xử lý request SOCKS5
func handleSOCKS5(clientConn net.Conn, pm *ProxyManager) {
	logger.Info("Handling SOCKS5 proxy request")
	defer clientConn.Close()

	// Đọc phiên bản SOCKS và số phương thức xác thực
	header := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, header); err != nil {
		logger.Error("Failed to read SOCKS5 header: %v", err)
		return
	}

	if header[0] != SOCKS5_VERSION {
		logger.Error("Unsupported SOCKS version: %d", header[0])
		clientConn.Write([]byte{SOCKS5_VERSION, 0xFF}) // 0xFF = Không phương thức nào được chấp nhận
		return
	}

	// Đọc danh sách phương thức xác thực được hỗ trợ
	methodCount := int(header[1])
	methods := make([]byte, methodCount)
	if _, err := io.ReadFull(clientConn, methods); err != nil {
		logger.Error("Failed to read authentication methods: %v", err)
		return
	}

	// Kiểm tra xem số lượng proxy SOCKS5 khả dụng
	socks5Selector := func(p *Proxy) bool {
		return p.Type == ProxyTypeSOCKS5
	}

	pm.mu.RLock()
	var socks5Count int
	for _, proxy := range pm.proxies {
		if proxy.IsWorking && socks5Selector(proxy) {
			socks5Count++
		}
	}
	pm.mu.RUnlock()

	logger.Info("Available SOCKS5 proxies: %d", socks5Count)

	// Hiện tại chúng ta chỉ hỗ trợ phương thức không xác thực (0)
	clientConn.Write([]byte{SOCKS5_VERSION, 0x00}) // Trả về: SOCKS5, phương thức 0 (không xác thực)

	// Đọc request
	header = make([]byte, 4)
	if _, err := io.ReadFull(clientConn, header); err != nil {
		logger.Error("Failed to read SOCKS5 request: %v", err)
		return
	}

	if header[0] != SOCKS5_VERSION {
		logger.Error("Unsupported SOCKS version in request: %d", header[0])
		return
	}

	if header[1] != SOCKS5_CMD_CONNECT {
		logger.Error("Unsupported SOCKS5 command: %d", header[1])
		// Trả về lỗi: phiên bản 5, lỗi 7 (Command not supported)
		clientConn.Write([]byte{SOCKS5_VERSION, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// Đọc địa chỉ
	var targetHost string
	var targetPort uint16

	addrType := header[3]
	switch addrType {
	case SOCKS5_ADDR_TYPE_IPV4:
		// Đọc IPv4 (4 bytes)
		addr := make([]byte, 4)
		if _, err := io.ReadFull(clientConn, addr); err != nil {
			logger.Error("Failed to read IPv4 address: %v", err)
			sendSocks5Error(clientConn, 0x01) // General server failure
			return
		}
		targetHost = net.IP(addr).String()

	case SOCKS5_ADDR_TYPE_DOMAIN:
		// Đọc độ dài tên miền
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(clientConn, lenByte); err != nil {
			logger.Error("Failed to read domain length: %v", err)
			sendSocks5Error(clientConn, 0x01)
			return
		}

		// Đọc tên miền
		domainLength := int(lenByte[0])
		if domainLength > 255 {
			logger.Error("Domain length too long: %d", domainLength)
			sendSocks5Error(clientConn, 0x01)
			return
		}

		domain := make([]byte, domainLength)
		if _, err := io.ReadFull(clientConn, domain); err != nil {
			logger.Error("Failed to read domain: %v", err)
			sendSocks5Error(clientConn, 0x01)
			return
		}
		targetHost = string(domain)

	case SOCKS5_ADDR_TYPE_IPV6:
		// Đọc IPv6 (16 bytes)
		addr := make([]byte, 16)
		if _, err := io.ReadFull(clientConn, addr); err != nil {
			logger.Error("Failed to read IPv6 address: %v", err)
			sendSocks5Error(clientConn, 0x01)
			return
		}
		targetHost = net.IP(addr).String()

	default:
		logger.Error("Unsupported address type: %d", addrType)
		sendSocks5Error(clientConn, 0x08) // Address type not supported
		return
	}

	// Đọc port (2 bytes)
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, portBytes); err != nil {
		logger.Error("Failed to read port: %v", err)
		sendSocks5Error(clientConn, 0x01)
		return
	}
	targetPort = binary.BigEndian.Uint16(portBytes)

	targetAddr := fmt.Sprintf("%s:%d", targetHost, targetPort)
	logger.Info("SOCKS5 target: %s", targetAddr)

	// Chọn proxy SOCKS5 để sử dụng
	socks5Selector = func(p *Proxy) bool {
		return p.Type == ProxyTypeSOCKS5
	}

	proxy := pm.GetRandomProxyWithFilter(socks5Selector)
	if proxy == nil {
		logger.Info("No available SOCKS5 proxies found, connecting directly to target")
		// Kết nối trực tiếp đến đích nếu không có proxy nào có sẵn
		targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
		if err != nil {
			logger.Error("Failed to connect directly to target: %v", err)
			sendSocks5Error(clientConn, 0x04) // Host unreachable
			return
		}
		defer targetConn.Close()

		// Gửi phản hồi thành công về client
		responseHeader := []byte{
			SOCKS5_VERSION,
			0x00, // Thành công
			0x00, // Reserved
			SOCKS5_ADDR_TYPE_IPV4,
			0, 0, 0, 0, // Bất kỳ địa chỉ IP nào
			0, 0, // Bất kỳ cổng nào
		}
		clientConn.Write(responseHeader)

		// Chuyển tiếp dữ liệu qua lại giữa client và target
		handleTLSOverSOCKS5(clientConn, "direct:"+targetAddr, targetConn)
		return
	}

	// Mở kết nối tới proxy SOCKS5
	logger.Info("Using SOCKS5 proxy: %s", proxy.URL)

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		logger.Error("Failed to parse proxy URL: %v", err)
		sendSocks5Error(clientConn, 0x01)
		return
	}

	// Lấy địa chỉ host:port từ URL
	proxyHost := proxyURL.Host
	if proxyHost == "" {
		// Nếu không có host trong URL (trường hợp không có schema)
		proxyHost = proxy.URL
	}

	logger.Info("Connecting to SOCKS5 proxy at %s", proxyHost)

	// Kết nối tới proxy
	proxyConn, err := net.DialTimeout("tcp", proxyHost, 10*time.Second)
	if err != nil {
		logger.Error("Failed to connect to SOCKS5 proxy: %v", err)
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}
	defer proxyConn.Close()

	// Thực hiện bắt tay SOCKS5 với proxy
	// 1. Xác định phương thức xác thực dựa trên credentials
	var authMethods []byte
	if proxy.Username != "" && proxy.Password != "" {
		// Hỗ trợ cả không xác thực (0) và user/pass (2)
		authMethods = []byte{0x00, 0x02}
	} else {
		// Chỉ hỗ trợ không xác thực (0)
		authMethods = []byte{0x00}
	}

	// Gửi phương thức xác thực
	auth := []byte{SOCKS5_VERSION, byte(len(authMethods))}
	auth = append(auth, authMethods...)
	if _, err := proxyConn.Write(auth); err != nil {
		logger.Error("Failed to send auth methods to proxy: %v", err)
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}

	// Đọc phản hồi
	authResp := make([]byte, 2)
	if _, err := io.ReadFull(proxyConn, authResp); err != nil {
		logger.Error("Failed to read auth response from proxy: %v", err)
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}

	if authResp[0] != SOCKS5_VERSION {
		logger.Error("Invalid SOCKS version in auth response: %v", authResp)
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}

	// Xử lý xác thực username/password nếu cần
	if authResp[1] == 0x02 && proxy.Username != "" && proxy.Password != "" {
		logger.Info("Proxy requires username/password authentication")

		// Gửi thông tin xác thực
		authReq := []byte{0x01} // Version 1 của protocol xác thực
		authReq = append(authReq, byte(len(proxy.Username)))
		authReq = append(authReq, []byte(proxy.Username)...)
		authReq = append(authReq, byte(len(proxy.Password)))
		authReq = append(authReq, []byte(proxy.Password)...)

		if _, err := proxyConn.Write(authReq); err != nil {
			logger.Error("Failed to send authentication to proxy: %v", err)
			sendSocks5Error(clientConn, 0x01)
			pm.MarkProxyFailed(proxy)
			return
		}

		// Đọc phản hồi xác thực
		authStatusResp := make([]byte, 2)
		if _, err := io.ReadFull(proxyConn, authStatusResp); err != nil {
			logger.Error("Failed to read auth status from proxy: %v", err)
			sendSocks5Error(clientConn, 0x01)
			pm.MarkProxyFailed(proxy)
			return
		}

		if authStatusResp[0] != 0x01 || authStatusResp[1] != 0x00 {
			logger.Error("Authentication failed: %v", authStatusResp)
			sendSocks5Error(clientConn, 0x01)
			pm.MarkProxyFailed(proxy)
			return
		}

		logger.Info("Authentication successful")
	} else if authResp[1] != 0x00 {
		logger.Error("Proxy did not accept authentication method: %v", authResp[1])
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}

	// 2. Gửi request kết nối
	// Xây dựng request: VER | CMD | RSV | ATYP | DST.ADDR | DST.PORT
	request := make([]byte, 0, 10+len(targetHost))
	request = append(request, SOCKS5_VERSION, SOCKS5_CMD_CONNECT, 0x00)

	// Thêm địa chỉ dựa vào loại
	switch addrType {
	case SOCKS5_ADDR_TYPE_IPV4:
		request = append(request, SOCKS5_ADDR_TYPE_IPV4)
		ip := net.ParseIP(targetHost).To4()
		request = append(request, ip...)

	case SOCKS5_ADDR_TYPE_DOMAIN:
		request = append(request, SOCKS5_ADDR_TYPE_DOMAIN)
		request = append(request, byte(len(targetHost)))
		request = append(request, []byte(targetHost)...)

	case SOCKS5_ADDR_TYPE_IPV6:
		request = append(request, SOCKS5_ADDR_TYPE_IPV6)
		ip := net.ParseIP(targetHost).To16()
		request = append(request, ip...)
	}

	// Thêm port
	request = append(request, byte(targetPort>>8), byte(targetPort))

	// Gửi request
	if _, err := proxyConn.Write(request); err != nil {
		logger.Error("Failed to send connection request to proxy: %v", err)
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}

	// Đọc phản hồi
	// Format: VER | REP | RSV | ATYP | BND.ADDR | BND.PORT
	reply := make([]byte, 4)
	if _, err := io.ReadFull(proxyConn, reply); err != nil {
		logger.Error("Failed to read connection response from proxy: %v", err)
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}

	if reply[0] != SOCKS5_VERSION {
		logger.Error("Invalid SOCKS version in response: %d", reply[0])
		sendSocks5Error(clientConn, 0x01)
		pm.MarkProxyFailed(proxy)
		return
	}

	if reply[1] != 0x00 {
		logger.Error("Proxy connection failed: %d", reply[1])
		// Chuyển tiếp mã lỗi từ proxy
		sendSocks5Error(clientConn, reply[1])
		pm.MarkProxyFailed(proxy)
		return
	}

	// Đọc phần còn lại của phản hồi (ATYP | BND.ADDR | BND.PORT)
	// Tùy thuộc vào loại địa chỉ
	switch reply[3] {
	case SOCKS5_ADDR_TYPE_IPV4:
		skipBytes := make([]byte, 4+2) // IP (4) + Port (2)
		io.ReadFull(proxyConn, skipBytes)

	case SOCKS5_ADDR_TYPE_DOMAIN:
		lenByte := make([]byte, 1)
		io.ReadFull(proxyConn, lenByte)
		skipBytes := make([]byte, int(lenByte[0])+2) // Domain + Port (2)
		io.ReadFull(proxyConn, skipBytes)

	case SOCKS5_ADDR_TYPE_IPV6:
		skipBytes := make([]byte, 16+2) // IP (16) + Port (2)
		io.ReadFull(proxyConn, skipBytes)
	}

	// Kết nối đã thành công, trả phản hồi thành công cho client
	// VER, REP, RSV, ATYP, BND.ADDR, BND.PORT
	// Sử dụng 0.0.0.0:0 làm địa chỉ bind
	successReply := []byte{
		SOCKS5_VERSION,
		0x00, // Succeeded
		0x00, // Reserved
		SOCKS5_ADDR_TYPE_IPV4,
		0, 0, 0, 0, // IP address (0.0.0.0)
		0, 0, // Port (0)
	}

	if _, err := clientConn.Write(successReply); err != nil {
		logger.Error("Failed to send success response to client: %v", err)
		return
	}

	// Đánh dấu proxy này thành công
	pm.MarkProxySuccess(proxy)

	// Tạo tunnel giữa client và target
	logger.Info("SOCKS5 connection established to %s via %s", targetAddr, proxy.URL)

	// Xử lý truyền dữ liệu hai chiều
	handleTLSOverSOCKS5(clientConn, targetAddr, proxyConn)
}

// handleTLSOverSOCKS5 xử lý kết nối TLS qua SOCKS5
func handleTLSOverSOCKS5(clientConn net.Conn, targetAddr string, proxyConn net.Conn) {
	// Tạo tunnel giữa client và proxy server
	logger.Info("SOCKS5 tunnel established to %s", targetAddr)

	// Kiểm tra cấu hình SOCKS5
	if socks5Config.SkipVerify {
		logger.Debug("SOCKS5 config: SSL verification skipped")
	}

	// Đối với SOCKS5, chúng ta không cần xử lý TLS
	// Chỉ cần tạo tunnel truyền dữ liệu hai chiều
	// Thư viện TLS của client sẽ tự xử lý handshake và verification

	errChan := make(chan error, 2)

	// Client -> Proxy
	go func() {
		_, err := io.Copy(proxyConn, clientConn)
		errChan <- err
	}()

	// Proxy -> Client
	go func() {
		_, err := io.Copy(clientConn, proxyConn)
		errChan <- err
	}()

	// Đợi một trong hai hướng kết thúc
	err := <-errChan
	if err != nil && err != io.EOF {
		logger.Error("SOCKS5 tunnel error: %v", err)
	}

	logger.Info("SOCKS5 connection to %s completed", targetAddr)
}

// sendSocks5Error gửi thông báo lỗi SOCKS5 cho client
func sendSocks5Error(conn net.Conn, errorCode byte) {
	// VER, REP, RSV, ATYP, BND.ADDR, BND.PORT
	errorReply := []byte{
		SOCKS5_VERSION,
		errorCode,
		0x00, // Reserved
		SOCKS5_ADDR_TYPE_IPV4,
		0, 0, 0, 0, // IP address (0.0.0.0)
		0, 0, // Port (0)
	}

	conn.Write(errorReply)
}
