#!/bin/bash
# Pi FM RDS Go - 依赖安装脚本
# T025: 安装脚本实现

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 权限运行此脚本"
        log_info "使用方法: sudo $0"
        exit 1
    fi
}

check_os() {
    if [ ! -f /etc/os-release ]; then
        log_error "无法检测操作系统版本"
        exit 1
    fi
    . /etc/os-release
    log_info "检测到操作系统: $PRETTY_NAME"
}

install_pifmrds() {
    log_info "检查 PiFmRds..."
    if command -v pi_fm_rds &> /dev/null; then
        log_info "PiFmRds 已安装: $(which pi_fm_rds)"
        return 0
    fi

    log_info "安装 PiFmRds 依赖..."
    apt-get update -qq
    apt-get install -y git build-essential libsndfile1-dev

    log_info "克隆 PiFmRds 仓库..."
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    git clone https://github.com/ChristopheJacquet/PiFmRds.git
    cd PiFmRds/src

    log_info "编译 PiFmRds..."
    make

    log_info "安装 PiFmRds..."
    cp pi_fm_rds /usr/local/bin/
    chmod +x /usr/local/bin/pi_fm_rds

    cd /
    rm -rf "$TEMP_DIR"
    log_info "PiFmRds 安装完成"
}

install_ffmpeg() {
    log_info "检查 FFmpeg..."
    if command -v ffmpeg &> /dev/null; then
        log_info "FFmpeg 已安装: $(ffmpeg -version | head -n1)"
        return 0
    fi

    log_info "安装 FFmpeg..."
    apt-get install -y ffmpeg
    log_info "FFmpeg 安装完成"
}

install_alsa() {
    log_info "检查 ALSA..."
    if command -v aplay &> /dev/null; then
        log_info "ALSA 已安装"
        return 0
    fi

    log_info "安装 ALSA..."
    apt-get install -y alsa-utils
    log_info "ALSA 安装完成"
}

configure_permissions() {
    log_info "配置权限..."

    # 创建必要的目录
    mkdir -p /var/log/pi-fm-rds-go
    mkdir -p /opt/pi-fm-rds-go/uploads
    mkdir -p /opt/pi-fm-rds-go/transcoded

    # 设置目录权限
    chown -R pi:pi /var/log/pi-fm-rds-go
    chown -R pi:pi /opt/pi-fm-rds-go
    chmod 755 /var/log/pi-fm-rds-go
    chmod 755 /opt/pi-fm-rds-go

    log_info "权限配置完成"
}

verify_installation() {
    log_info "验证安装..."

    local errors=0

    if ! command -v pi_fm_rds &> /dev/null; then
        log_error "PiFmRds 未正确安装"
        errors=$((errors + 1))
    fi

    if ! command -v ffmpeg &> /dev/null; then
        log_error "FFmpeg 未正确安装"
        errors=$((errors + 1))
    fi

    if ! command -v aplay &> /dev/null; then
        log_error "ALSA 未正确安装"
        errors=$((errors + 1))
    fi

    if [ $errors -eq 0 ]; then
        log_info "所有依赖安装成功！"
        return 0
    else
        log_error "安装验证失败，发现 $errors 个错误"
        return 1
    fi
}

main() {
    log_info "开始安装 Pi FM RDS Go 依赖..."

    check_root
    check_os

    install_pifmrds
    install_ffmpeg
    install_alsa
    configure_permissions

    if verify_installation; then
        log_info "安装完成！"
        log_info "下一步: 运行 ./scripts/setup-sudo.sh 配置 sudo 权限"
        exit 0
    else
        log_error "安装失败"
        exit 1
    fi
}

main "$@"
