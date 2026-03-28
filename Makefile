.PHONY: all build build-pi test test-coverage clean run help

# 默认目标
all: test build

# 构建二进制文件
build:
	@echo "Building pi-fm-rds-go..."
	GOROOT="" go build -o bin/pi-fm-rds-go cmd/server/main.go

# 交叉编译（树莓派 ARM64）
build-pi:
	@echo "Building for Raspberry Pi (ARM64)..."
	GOROOT="" GOOS=linux GOARCH=arm64 go build -o bin/pi-fm-rds-go-arm64 cmd/server/main.go

# 运行测试
test:
	@echo "Running tests..."
	GOROOT="" go test -v ./...

# 运行测试并显示覆盖率
test-coverage:
	@echo "Running tests with coverage..."
	GOROOT="" go test -v -cover ./...
	@echo "\nGenerating coverage report..."
	GOROOT="" go test -coverprofile=coverage.out ./...
	GOROOT="" go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 清理构建产物
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# 运行服务器
run:
	@echo "Starting server..."
	GOROOT="" go run cmd/server/main.go

# 创建必要的目录
setup:
	@echo "Creating directories..."
	mkdir -p uploads transcoded bin

# 显示帮助信息
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make build-pi       - Cross-compile for Raspberry Pi (ARM64)"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run            - Run the server"
	@echo "  make setup          - Create necessary directories"
	@echo "  make help           - Show this help message"
