#!/bin/bash

# Màu sắc cho terminal
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
RED='\033[0;31m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== PROXY SERVER WATCHER & TESTER ===${NC}"
echo -e "${YELLOW}Watching for changes in main.go and proxy directory${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}\n"

LAST_MODIFIED_TIME=0
SERVER_PID=""
RUN_TESTS=true  # Tùy chọn chạy test tự động
PORT=8080       # Cổng mặc định

# Xử lý tham số dòng lệnh
for arg in "$@"; do
  case $arg in
    --no-test)
      RUN_TESTS=false
      shift
      ;;
    --port=*)
      PORT="${arg#*=}"
      shift
      ;;
    *)
      # Tham số không rõ
      echo -e "${RED}Tham số không được hỗ trợ: $arg${NC}"
      echo -e "Các tham số được hỗ trợ: --no-test, --port=PORT"
      exit 1
      ;;
  esac
done

# Hàm để kiểm tra sự thay đổi của file
check_changes() {
    local current_time=$(find . -type f \( -name "main.go" -o -path "./proxy/*.go" \) -exec stat -f "%m" {} \; | sort -nr | head -n 1)
    
    if [ "$current_time" != "$LAST_MODIFIED_TIME" ]; then
        return 0 # Có thay đổi
    else
        return 1 # Không có thay đổi
    fi
}

# Hàm để chạy test
run_tests() {
    if [ "$RUN_TESTS" = true ]; then
        echo -e "${PURPLE}Chạy tests...${NC}"
        sleep 2 # Đợi server khởi động hoàn tất
        go run test/checkip.go --proxy="http://localhost:$PORT"
    fi
}

# Hàm để khởi động lại server
restart_server() {
    echo -e "${YELLOW}$(date '+%H:%M:%S') ${BLUE}Phát hiện thay đổi, khởi động lại server...${NC}"
    
    # Dừng server cũ nếu có
    if [ ! -z "$SERVER_PID" ]; then
        echo -e "${YELLOW}Dừng server với PID: $SERVER_PID${NC}"
        kill $SERVER_PID 2>/dev/null
        sleep 1
        
        # Nếu vẫn chạy, buộc dừng
        if ps -p $SERVER_PID > /dev/null; then
            echo -e "${YELLOW}Server không dừng, buộc dừng...${NC}"
            kill -9 $SERVER_PID 2>/dev/null
            sleep 1
        fi
    fi
    
    # Biên dịch để kiểm tra lỗi
    echo -e "${BLUE}Biên dịch để kiểm tra lỗi...${NC}"
    go build -o /tmp/proxy-server main.go 2>/tmp/build_errors.log
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Lỗi biên dịch:${NC}"
        cat /tmp/build_errors.log
        return 1
    fi
    
    # Khởi động server mới
    echo -e "${GREEN}Khởi động server trên cổng $PORT...${NC}"
    go run main.go > /tmp/proxy-server.log 2>&1 &
    SERVER_PID=$!
    
    # Kiểm tra server đã chạy chưa
    sleep 1
    if ! ps -p $SERVER_PID > /dev/null; then
        echo -e "${RED}Lỗi khởi động server! Kiểm tra log:${NC}"
        cat /tmp/proxy-server.log
        return 1
    fi
    
    echo -e "${GREEN}Server đã khởi động với PID: $SERVER_PID${NC}"
    
    # Lưu thời gian sửa đổi mới nhất
    LAST_MODIFIED_TIME=$(find . -type f \( -name "main.go" -o -path "./proxy/*.go" \) -exec stat -f "%m" {} \; | sort -nr | head -n 1)
    
    # Chạy tests nếu được yêu cầu
    run_tests
    
    return 0
}

# Xử lý khi nhận tín hiệu thoát
cleanup() {
    echo -e "\n${YELLOW}Dừng proxy server...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null
    fi
    echo -e "${GREEN}Đã dừng. Tạm biệt!${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Khởi động server lần đầu
restart_server || exit 1

# Vòng lặp chính
while true; do
    sleep 1
    
    # Kiểm tra xem server có đang chạy không
    if ! ps -p $SERVER_PID > /dev/null; then
        echo -e "${YELLOW}Server đã dừng. Đang khởi động lại...${NC}"
        restart_server || {
            echo -e "${RED}Không thể khởi động lại server sau lỗi. Đợi thay đổi code...${NC}"
            sleep 5
            continue
        }
    fi
    
    # Kiểm tra sự thay đổi
    if check_changes; then
        restart_server
    fi
done 