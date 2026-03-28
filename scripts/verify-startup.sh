#!/bin/bash
# Pi FM RDS Go - 启动验证脚本
# T028: 启动验证实现 (FR-008, FR-009)

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_dependencies() {
    log_info "检查依赖..."
    local errors=0

    # FR-008: 依赖检查
    if ! command -v pi_fm_rds &> /dev/null; then
        log_error "未找到 PiFmRds"
        log_info "  解决方法: 运行 sudo ./scripts/install.sh"
        errors=$((errors + 1))
    else
        log_info "✓ PiFmRds: $(which pi_fm_rds)"
    fi

    if ! command -v ffmpeg &> /dev/null; then
        log_error "未找到 FFmpeg"
        log_info "  解决方法: sudo apt-get install ffmpeg"
        errors=$((errors + 1))
    else
        log_info "✓ FFmpeg: $(ffmpeg -version | head -n1)"
    fi

    if ! command -v aplay &> /dev/null; then
        log_error "未找到 ALSA"
        log_info "  解决方法: sudo apt-get install alsa-utils"
        errors=$((errors + 1))
    else
        log_info "✓ ALSA: $(which aplay)"
    fi

    return $errors
}

check_gpio() {
    log_info "检查 GPIO 权限..."

    # FR-009: GPIO 检查
    if [ ! -d /sys/class/gpio ]; then
        log_error "未找到 GPIO 设备"
        log_info "  此程序需要在树莓派上运行"
        return 1
    fi

    log_info "✓ GPIO 设备可用"

    # 检查 GPIO 4 (默认 FM 输出引脚)
    if [ -e /sys/class/gpio/gpio4 ]; then
        log_info "✓ GPIO 4 已导出"
    else
        log_warn "GPIO 4 未导出 (首次运行时会自动配置)"
    fi

    return 0
}

check_sudo_config() {
    log_info "检查 sudo 配置..."

    if [ -f /etc/sudoers.d/pi-fm-rds ]; then
        log_info "✓ sudo 配置文件存在"

        if sudo -n pi_fm_rds --help &> /dev/null 2>&1; then
            log_info "✓ sudo 权限配置正确"
        else
            log_warn "sudo 权限可能未生效"
            log_info "  解决方法: 运行 sudo ./scripts/setup-sudo.sh"
        fi
    else
        log_error "未找到 sudo 配置"
        log_info "  解决方法: 运行 sudo ./scripts/setup-sudo.sh"
        return 1
    fi

    return 0
}

check_directories() {
    log_info "检查目录..."

    local dirs=(
        "/opt/pi-fm-rds-go"
        "/opt/pi-fm-rds-go/uploads"
        "/opt/pi-fm-rds-go/transcoded"
        "/var/log/pi-fm-rds-go"
    )

    for dir in "${dirs[@]}"; do
        if [ -d "$dir" ]; then
            log_info "✓ $dir"
        else
            log_warn "目录不存在: $dir"
            log_info "  解决方法: sudo mkdir -p $dir && sudo chown pi:pi $dir"
        fi
    done

    return 0
}

check_config() {
    log_info "检查配置文件..."

    if [ -f "config.yaml" ]; then
        log_info "✓ 配置文件存在: config.yaml"
    else
        log_error "未找到配置文件"
        log_info "  解决方法: 复制 config.yaml.example 到 config.yaml"
        return 1
    fi

    return 0
}

main() {
    log_info "Pi FM RDS Go - 启动验证"
    echo ""

    local total_errors=0

    check_dependencies || total_errors=$((total_errors + $?))
    echo ""

    check_gpio || total_errors=$((total_errors + $?))
    echo ""

    check_sudo_config || total_errors=$((total_errors + $?))
    echo ""

    check_directories || total_errors=$((total_errors + $?))
    echo ""

    check_config || total_errors=$((total_errors + $?))
    echo ""

    if [ $total_errors -eq 0 ]; then
        log_info "所有检查通过！系统已准备就绪"
        exit 0
    else
        log_error "发现 $total_errors 个问题，请按照上述提示解决"
        exit 1
    fi
}

main "$@"
