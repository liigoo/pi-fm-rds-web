#!/bin/bash
# Pi FM RDS Go - Sudo 配置脚本
# T027: Sudo 配置实现

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

setup_sudo() {
    log_info "配置 sudo 权限..."

    SUDOERS_FILE="/etc/sudoers.d/pi-fm-rds"
    PIFMRDS_PATH=$(which pi_fm_rds 2>/dev/null || echo "/usr/local/bin/pi_fm_rds")

    if [ ! -f "$PIFMRDS_PATH" ]; then
        log_error "未找到 pi_fm_rds，请先运行 install.sh"
        exit 1
    fi

    log_info "创建 sudoers 配置文件..."
    cat > "$SUDOERS_FILE" << EOF
# Pi FM RDS Go - 最小权限配置
# 允许 pi 用户无密码运行 pi_fm_rds
pi ALL=(ALL) NOPASSWD: $PIFMRDS_PATH
EOF

    chmod 0440 "$SUDOERS_FILE"
    log_info "sudoers 配置文件已创建: $SUDOERS_FILE"

    if visudo -c -f "$SUDOERS_FILE" &> /dev/null; then
        log_info "sudoers 配置验证成功"
    else
        log_error "sudoers 配置验证失败"
        rm -f "$SUDOERS_FILE"
        exit 1
    fi
}

verify_sudo() {
    log_info "验证 sudo 配置..."

    if sudo -n -u pi sudo -n pi_fm_rds --help &> /dev/null; then
        log_info "sudo 配置验证成功"
        return 0
    else
        log_warn "sudo 配置可能未生效，请重新登录后测试"
        return 0
    fi
}

main() {
    log_info "开始配置 Pi FM RDS Go sudo 权限..."

    check_root
    setup_sudo
    verify_sudo

    log_info "配置完成！"
    log_info "下一步: 配置 systemd 服务"
    exit 0
}

main "$@"
