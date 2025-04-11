# Proxy Server

A high-performance proxy server with automatic proxy rotation and authentication support.

## Features

- Automatic proxy rotation
- Support for both HTTP and HTTPS requests
- Proxy authentication support
- Clean and efficient codebase
- Detailed logging

## Requirements

- Go 1.24 or higher
- Access to proxy servers

## Installation

1. Clone the repository:
```bash
git clone https://github.com/manhgdev/proxy-server.git
cd proxy-server
```

2. Install dependencies:
```bash
go mod download
```

## Configuration

1. Create a `proxy_list.txt` file in the root directory with your proxy list in the following format:
```
ip:port:username:password
```

Example:
```
1.2.3.4:8080
1.2.3.4:8080:admin:123456
admin:123456@1.2.3.4:8080
proxy.example.com:8080
admin:123456@proxy.example.com:8080 
```

## Usage

1. Start the server:
```bash
go run main.go
```

2. Use the proxy server:
```bash
curl -x http://localhost:8080 https://api.zm.io.vn/check-ip
curl -x http://localhost:8080 http://ip4.me/api/
curl -x http://localhost:8080 ip4.me/api
```

## Project Structure

```
.
├── main.go              # Main entry point
├── proxy_list.txt       # List of proxy servers
├── proxy/
│   ├── server.go        # Proxy server implementation
│   └── manager.go       # Proxy manager implementation
└── utils/
    └── logger.go        # Logging utilities
```

## License

MIT License 