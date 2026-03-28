# Pi FM RDS Go

基于 Go 语言的树莓派 FM 广播系统，支持 RDS 功能和 Web 管理界面。

## 项目结构

```
pi-fm-rds-go/
├── cmd/
│   └── server/          # 主程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── process/         # 进程管理
│   ├── audio/           # 音频处理
│   ├── playlist/        # 播放列表管理
│   ├── websocket/       # WebSocket 服务
│   ├── api/             # HTTP API
│   ├── storage/         # 存储管理
│   └── errors/          # 错误处理
├── web/
│   ├── static/          # 静态资源
│   └── templates/       # HTML 模板
├── uploads/             # 上传文件目录
├── transcoded/          # 转码文件目录
├── scripts/             # 脚本文件
├── systemd/             # systemd 服务配置
└── config.yaml          # 配置文件
```

## 快速开始

### 安装依赖

```bash
go mod download
```

### 配置

编辑 `config.yaml` 文件：

```yaml
server:
  port: 8080
  host: "0.0.0.0"

pifmrds:
  binary_path: "/usr/local/bin/pi_fm_rds"
  default_frequency: 107.9

storage:
  upload_dir: "./uploads"
  transcoded_dir: "./transcoded"
  max_file_size: 104857600  # 100MB
  max_total_size: 2147483648  # 2GB

audio:
  sample_rate: 22050
  channels: 1

websocket:
  max_clients: 5
  spectrum_fps: 10
```

### 环境变量覆盖

支持通过环境变量覆盖配置：

- `SERVER_PORT` - 服务器端口
- `SERVER_HOST` - 服务器地址
- `PIFMRDS_BINARY_PATH` - pi_fm_rds 二进制文件路径
- `PIFMRDS_DEFAULT_FREQUENCY` - 默认 FM 频率

### 运行

```bash
# 使用默认配置
go run cmd/server/main.go

# 指定配置文件
go run cmd/server/main.go -config /path/to/config.yaml

# 使用环境变量覆盖
SERVER_PORT=9090 go run cmd/server/main.go
```

## 开发

### 运行测试

```bash
# 运行所有测试
make test

# 运行测试并显示覆盖率
make test-coverage

# 运行特定模块测试
go test -v ./internal/config/
```

### 构建

```bash
# 构建二进制文件
make build

# 交叉编译（树莓派）
make build-pi
```

## 测试覆盖率

当前测试覆盖率：80.4%

## 许可证

MIT License
