#!/bin/bash
#
# Print Service Installation Script for macOS
#
# This script installs the ERP Print Service as a launchd service.
# Run with sudo/root privileges for system-wide installation,
# or without sudo for user-level installation.
#
# Usage: ./install-mac.sh [OPTIONS]
#
# Options:
#   --prefix DIR       Installation prefix (default: /usr/local/print-service or ~/Library/PrintService)
#   --system           Install as system-wide service (requires sudo)
#   --user             Install as user-level service (default)
#   --config FILE      Path to config file (default: PREFIX/config.yaml)
#   --uninstall        Uninstall the service
#   --help             Show this help message
#

set -e

# Default values
PREFIX=""
SYSTEM_INSTALL=false
CONFIG_FILE=""
UNINSTALL=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="print-service"
PLIST_NAME="com.erp.print-service"

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
    sed -n '2,18p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prefix)
                PREFIX="$2"
                shift 2
                ;;
            --system)
                SYSTEM_INSTALL=true
                shift
                ;;
            --user)
                SYSTEM_INSTALL=false
                shift
                ;;
            --config)
                CONFIG_FILE="$2"
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

    # Set default prefix based on installation type
    if [[ -z "$PREFIX" ]]; then
        if [[ "$SYSTEM_INSTALL" == true ]]; then
            PREFIX="/usr/local/print-service"
        else
            PREFIX="${HOME}/Library/PrintService"
        fi
    fi

    # Set default config file if not specified
    if [[ -z "$CONFIG_FILE" ]]; then
        CONFIG_FILE="${PREFIX}/config.yaml"
    fi
}

# Get plist path based on installation type
get_plist_path() {
    if [[ "$SYSTEM_INSTALL" == true ]]; then
        echo "/Library/LaunchDaemons/${PLIST_NAME}.plist"
    else
        echo "${HOME}/Library/LaunchAgents/${PLIST_NAME}.plist"
    fi
}

# Check for required commands
check_requirements() {
    if ! command -v launchctl &> /dev/null; then
        log_error "launchctl not found. This script requires macOS."
        exit 1
    fi

    if [[ "$SYSTEM_INSTALL" == true ]] && [[ $EUID -ne 0 ]]; then
        log_error "System-wide installation requires root privileges (use sudo)"
        exit 1
    fi
}

# Detect architecture and find binary
find_binary() {
    local arch=$(uname -m)
    local binary_suffix=""

    if [[ "$arch" == "arm64" ]]; then
        binary_suffix="darwin-arm64"
    else
        binary_suffix="darwin-amd64"
    fi

    # Search for binary in various locations
    local search_paths=(
        "${SCRIPT_DIR}/${BINARY_NAME}"
        "${SCRIPT_DIR}/${BINARY_NAME}-${binary_suffix}"
        "${SCRIPT_DIR}/../dist/${BINARY_NAME}-${binary_suffix}"
    )

    for path in "${search_paths[@]}"; do
        if [[ -f "$path" ]]; then
            echo "$path"
            return 0
        fi
    done

    log_error "Binary not found for ${arch}. Searched:"
    for path in "${search_paths[@]}"; do
        log_error "  - $path"
    done
    exit 1
}

# Install binary and configuration
install_files() {
    log_info "Installing files to ${PREFIX}..."

    # Create directories
    mkdir -p "${PREFIX}"
    mkdir -p "${PREFIX}/logs"

    # Find and copy binary
    local binary_src=$(find_binary)
    cp "$binary_src" "${PREFIX}/${BINARY_NAME}"
    chmod 755 "${PREFIX}/${BINARY_NAME}"
    log_info "Installed binary: ${PREFIX}/${BINARY_NAME}"

    # Verify the binary works
    if ! "${PREFIX}/${BINARY_NAME}" -version &>/dev/null; then
        log_error "Binary verification failed. The binary may not be compatible with this system."
        exit 1
    fi

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

    log_info "Files installed successfully"
}

# Create launchd plist
create_launchd_plist() {
    log_info "Creating launchd service..."

    local plist_path=$(get_plist_path)
    local plist_dir=$(dirname "$plist_path")

    # Create directory if needed
    mkdir -p "$plist_dir"

    # Note: System-wide services don't specify UserName/GroupName
    # The service will run as root by default for system daemons
    # For better security, consider using user-level installation

    cat > "$plist_path" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${PLIST_NAME}</string>

    <key>ProgramArguments</key>
    <array>
        <string>${PREFIX}/${BINARY_NAME}</string>
        <string>-config</string>
        <string>${CONFIG_FILE}</string>
    </array>

    <key>WorkingDirectory</key>
    <string>${PREFIX}</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>StandardOutPath</key>
    <string>${PREFIX}/logs/stdout.log</string>

    <key>StandardErrorPath</key>
    <string>${PREFIX}/logs/stderr.log</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PRINT_SERVICE_LOG_DIR</key>
        <string>${PREFIX}/logs</string>
    </dict>

    <key>ThrottleInterval</key>
    <integer>5</integer>

    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>
EOF

    chmod 644 "$plist_path"
    log_info "Created launchd plist: $plist_path"
}

# Load and start service
start_service() {
    log_info "Loading and starting service..."

    local plist_path=$(get_plist_path)

    # Unload if already loaded
    if launchctl list | grep -q "$PLIST_NAME" 2>/dev/null; then
        if [[ "$SYSTEM_INSTALL" == true ]]; then
            sudo launchctl unload "$plist_path" 2>/dev/null || true
        else
            launchctl unload "$plist_path" 2>/dev/null || true
        fi
    fi

    # Load the service
    if [[ "$SYSTEM_INSTALL" == true ]]; then
        sudo launchctl load "$plist_path"
    else
        launchctl load "$plist_path"
    fi

    sleep 2

    # Check if service is running
    if launchctl list | grep -q "$PLIST_NAME"; then
        log_info "Service loaded successfully"

        # Wait for health endpoint
        local max_attempts=10
        local attempt=1
        while [[ $attempt -le $max_attempts ]]; do
            if curl -s http://localhost:9999/health &>/dev/null; then
                log_info "Service is running and healthy"
                curl -s http://localhost:9999/health | python3 -m json.tool 2>/dev/null || true
                return 0
            fi
            sleep 1
            ((attempt++))
        done

        log_warn "Service loaded but health check not responding"
        log_warn "Check logs: tail -f ${PREFIX}/logs/stderr.log"
    else
        log_warn "Service failed to load. Check configuration and logs:"
        log_warn "  tail -f ${PREFIX}/logs/stderr.log"
    fi
}

# Uninstall service
uninstall() {
    log_info "Uninstalling print service..."

    local plist_path=$(get_plist_path)

    # Unload service
    if launchctl list | grep -q "$PLIST_NAME" 2>/dev/null; then
        if [[ "$SYSTEM_INSTALL" == true ]]; then
            sudo launchctl unload "$plist_path" 2>/dev/null || true
        else
            launchctl unload "$plist_path" 2>/dev/null || true
        fi
        log_info "Unloaded service"
    fi

    # Remove plist
    if [[ -f "$plist_path" ]]; then
        if [[ "$SYSTEM_INSTALL" == true ]]; then
            sudo rm -f "$plist_path"
        else
            rm -f "$plist_path"
        fi
        log_info "Removed launchd plist"
    fi

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

    log_info "Uninstall complete"
}

# Print installation summary
print_summary() {
    local plist_path=$(get_plist_path)
    local install_type="user-level"
    if [[ "$SYSTEM_INSTALL" == true ]]; then
        install_type="system-wide"
    fi

    echo ""
    echo "========================================"
    echo "  Print Service Installation Complete"
    echo "========================================"
    echo ""
    echo "Installation type:      ${install_type}"
    echo "Installation directory: ${PREFIX}"
    echo "Configuration file:     ${CONFIG_FILE}"
    echo "Launchd plist:          ${plist_path}"
    echo ""
    echo "Service commands:"
    if [[ "$SYSTEM_INSTALL" == true ]]; then
        echo "  Load:    sudo launchctl load ${plist_path}"
        echo "  Unload:  sudo launchctl unload ${plist_path}"
    else
        echo "  Load:    launchctl load ${plist_path}"
        echo "  Unload:  launchctl unload ${plist_path}"
    fi
    echo "  Status:  launchctl list | grep print-service"
    echo "  Logs:    tail -f ${PREFIX}/logs/stderr.log"
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

    if [[ "$UNINSTALL" == true ]]; then
        uninstall
        exit 0
    fi

    check_requirements
    install_files
    create_launchd_plist
    start_service
    print_summary
}

main "$@"
