#!/bin/bash
#
# Print Service Installation Script for Linux
#
# This script installs the ERP Print Service as a systemd service.
# Run with sudo/root privileges.
#
# Usage: sudo ./install.sh [OPTIONS]
#
# Options:
#   --prefix DIR       Installation prefix (default: /opt/print-service)
#   --user USER        Service user (default: print-service)
#   --group GROUP      Service group (default: print-service)
#   --config FILE      Path to config file (default: PREFIX/config.yaml)
#   --uninstall        Uninstall the service
#   --help             Show this help message
#

set -e

# Default values
PREFIX="/opt/print-service"
SERVICE_USER="print-service"
SERVICE_GROUP="print-service"
CONFIG_FILE=""
UNINSTALL=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="print-service"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Show help message
show_help() {
    sed -n '2,16p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prefix)
                PREFIX="$2"
                # Validate prefix is an absolute path
                if [[ ! "$PREFIX" =~ ^/ ]]; then
                    log_error "Prefix must be an absolute path (starting with /)"
                    exit 1
                fi
                # Prevent path traversal
                if [[ "$PREFIX" =~ \.\. ]]; then
                    log_error "Prefix cannot contain .."
                    exit 1
                fi
                shift 2
                ;;
            --user)
                SERVICE_USER="$2"
                # Validate username format
                if [[ ! "$SERVICE_USER" =~ ^[a-z_][a-z0-9_-]*$ ]]; then
                    log_error "Invalid username format: $SERVICE_USER"
                    exit 1
                fi
                shift 2
                ;;
            --group)
                SERVICE_GROUP="$2"
                # Validate group format
                if [[ ! "$SERVICE_GROUP" =~ ^[a-z_][a-z0-9_-]*$ ]]; then
                    log_error "Invalid group format: $SERVICE_GROUP"
                    exit 1
                fi
                shift 2
                ;;
            --config)
                CONFIG_FILE="$2"
                # Validate config path is absolute
                if [[ ! "$CONFIG_FILE" =~ ^/ ]]; then
                    log_error "Config path must be absolute (starting with /)"
                    exit 1
                fi
                shift 2
                ;;
            --uninstall)
                UNINSTALL=true
                shift
                ;;
            --help|-h)
                show_help
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                ;;
        esac
    done

    # Set default config file if not specified
    if [[ -z "$CONFIG_FILE" ]]; then
        CONFIG_FILE="${PREFIX}/config.yaml"
    fi
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check for required commands
check_requirements() {
    local missing=()

    for cmd in systemctl useradd groupadd; do
        if ! command -v "$cmd" &> /dev/null; then
            missing+=("$cmd")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required commands: ${missing[*]}"
        log_error "This script requires a systemd-based Linux distribution"
        exit 1
    fi
}

# Create service user and group
create_user() {
    log_info "Creating service user and group..."

    # Create group if it doesn't exist
    if ! getent group "$SERVICE_GROUP" &>/dev/null; then
        groupadd --system "$SERVICE_GROUP"
        log_info "Created group: $SERVICE_GROUP"
    else
        log_info "Group already exists: $SERVICE_GROUP"
    fi

    # Create user if it doesn't exist
    if ! id "$SERVICE_USER" &>/dev/null; then
        useradd --system \
            --gid "$SERVICE_GROUP" \
            --shell /usr/sbin/nologin \
            --home-dir "$PREFIX" \
            --no-create-home \
            "$SERVICE_USER"
        log_info "Created user: $SERVICE_USER"
    else
        log_info "User already exists: $SERVICE_USER"
    fi
}

# Install binary and configuration
install_files() {
    log_info "Installing files to ${PREFIX}..."

    # Create directories
    mkdir -p "${PREFIX}"
    mkdir -p "${PREFIX}/logs"

    # Find and copy binary
    local binary_src=""
    if [[ -f "${SCRIPT_DIR}/${BINARY_NAME}" ]]; then
        binary_src="${SCRIPT_DIR}/${BINARY_NAME}"
    elif [[ -f "${SCRIPT_DIR}/../dist/${BINARY_NAME}-linux-amd64" ]]; then
        binary_src="${SCRIPT_DIR}/../dist/${BINARY_NAME}-linux-amd64"
    elif [[ -f "${SCRIPT_DIR}/${BINARY_NAME}-linux-amd64" ]]; then
        binary_src="${SCRIPT_DIR}/${BINARY_NAME}-linux-amd64"
    else
        log_error "Binary not found. Please ensure ${BINARY_NAME} is in the same directory."
        exit 1
    fi

    cp "$binary_src" "${PREFIX}/${BINARY_NAME}"
    chmod 755 "${PREFIX}/${BINARY_NAME}"
    log_info "Installed binary: ${PREFIX}/${BINARY_NAME}"

    # Copy configuration if it doesn't exist
    if [[ ! -f "$CONFIG_FILE" ]]; then
        local config_src=""
        if [[ -f "${SCRIPT_DIR}/config.example.yaml" ]]; then
            config_src="${SCRIPT_DIR}/config.example.yaml"
        elif [[ -f "${SCRIPT_DIR}/../configs/config.example.yaml" ]]; then
            config_src="${SCRIPT_DIR}/../configs/config.example.yaml"
        fi

        if [[ -n "$config_src" ]]; then
            cp "$config_src" "$CONFIG_FILE"
            chmod 600 "$CONFIG_FILE"
            log_info "Created configuration: $CONFIG_FILE"
            log_warn "Please edit $CONFIG_FILE with your settings"
        else
            log_warn "No config template found. You need to create $CONFIG_FILE manually."
        fi
    else
        log_info "Configuration already exists: $CONFIG_FILE"
    fi

    # Set ownership
    chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "${PREFIX}"

    log_info "Files installed successfully"
}

# Create systemd service
create_systemd_service() {
    log_info "Creating systemd service..."

    local service_file="/etc/systemd/system/print-service.service"

    cat > "$service_file" << EOF
[Unit]
Description=ERP Print Service
Documentation=https://github.com/example/erp/tree/main/tools/print-service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
ExecStart=${PREFIX}/${BINARY_NAME} -config ${CONFIG_FILE}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=print-service

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictNamespaces=true

# Allow write to logs directory
ReadWritePaths=${PREFIX}/logs

# Environment
Environment=PRINT_SERVICE_LOG_DIR=${PREFIX}/logs

# Resource limits
LimitNOFILE=65536
MemoryMax=512M
CPUQuota=100%

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "$service_file"
    log_info "Created systemd service: $service_file"

    # Reload systemd
    systemctl daemon-reload
    log_info "Reloaded systemd configuration"
}

# Enable and start service
start_service() {
    log_info "Enabling and starting service..."

    systemctl enable print-service
    log_info "Service enabled"

    # Only start if config exists and is valid
    if [[ -f "$CONFIG_FILE" ]]; then
        systemctl start print-service
        sleep 2

        if systemctl is-active --quiet print-service; then
            log_info "Service started successfully"
            systemctl status print-service --no-pager
        else
            log_warn "Service failed to start. Check configuration and logs:"
            log_warn "  journalctl -u print-service -n 50"
        fi
    else
        log_warn "Configuration file not found. Service not started."
        log_warn "Create $CONFIG_FILE and run: sudo systemctl start print-service"
    fi
}

# Uninstall service
uninstall() {
    log_info "Uninstalling print service..."

    # Stop and disable service
    if systemctl is-active --quiet print-service; then
        systemctl stop print-service
        log_info "Stopped service"
    fi

    if systemctl is-enabled --quiet print-service 2>/dev/null; then
        systemctl disable print-service
        log_info "Disabled service"
    fi

    # Remove service file
    rm -f /etc/systemd/system/print-service.service
    systemctl daemon-reload
    log_info "Removed systemd service"

    # Ask about removing files
    echo ""
    read -p "Remove installation directory ${PREFIX}? [y/N] " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "${PREFIX}"
        log_info "Removed ${PREFIX}"
    else
        log_info "Kept ${PREFIX}"
    fi

    # Ask about removing user
    if id "$SERVICE_USER" &>/dev/null; then
        read -p "Remove service user ${SERVICE_USER}? [y/N] " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            userdel "$SERVICE_USER"
            log_info "Removed user: $SERVICE_USER"
        fi
    fi

    log_info "Uninstall complete"
}

# Print installation summary
print_summary() {
    echo ""
    echo "========================================"
    echo "  Print Service Installation Complete"
    echo "========================================"
    echo ""
    echo "Installation directory: ${PREFIX}"
    echo "Configuration file:     ${CONFIG_FILE}"
    echo "Service user:           ${SERVICE_USER}"
    echo "Service group:          ${SERVICE_GROUP}"
    echo ""
    echo "Service commands:"
    echo "  Start:   sudo systemctl start print-service"
    echo "  Stop:    sudo systemctl stop print-service"
    echo "  Status:  sudo systemctl status print-service"
    echo "  Logs:    sudo journalctl -u print-service -f"
    echo ""
    echo "Health check:"
    echo "  curl http://localhost:9999/health"
    echo ""
    echo "Version:"
    "${PREFIX}/${BINARY_NAME}" -version
    echo ""
}

# Main installation flow
main() {
    parse_args "$@"
    check_root

    if [[ "$UNINSTALL" == true ]]; then
        uninstall
        exit 0
    fi

    check_requirements
    create_user
    install_files
    create_systemd_service
    start_service
    print_summary
}

main "$@"
