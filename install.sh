#!/bin/bash

#
# Copyright (c) 2025 Red Hat, Inc.
# This program and the accompanying materials are made
# available under the terms of the Eclipse Public License 2.0
# which is available at https://www.eclipse.org/legal/epl-2.0/
#
# SPDX-License-Identifier: EPL-2.0
#
# Contributors:
#   Red Hat, Inc. - initial API and implementation
#

# kubectx-manager Installation Script
# This script downloads and installs the latest version of kubectx-manager

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="che-incubator/kubectx-manager"
BINARY_NAME="kubectx-manager"
INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/bin"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *)          log_error "Unsupported operating system: $(uname -s)" && exit 1 ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)              log_error "Unsupported architecture: $(uname -m)" && exit 1 ;;
    esac
    
    echo "${os}_${arch}"
}

# Get latest release version from GitHub API
get_latest_version() {
    log_info "Fetching latest release information..."
    
    if command -v curl >/dev/null 2>&1; then
        curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        log_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
}

# Download and extract binary
download_binary() {
    local version="$1"
    local platform="$2"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${version#v}_${platform}.tar.gz"
    local temp_dir=$(mktemp -d)
    local archive_file="${temp_dir}/${BINARY_NAME}.tar.gz"
    
    log_info "Downloading ${BINARY_NAME} ${version} for ${platform}..."
    log_info "URL: ${download_url}"
    
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL "${download_url}" -o "${archive_file}"; then
            log_error "Failed to download ${BINARY_NAME}"
            rm -rf "${temp_dir}"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "${download_url}" -O "${archive_file}"; then
            log_error "Failed to download ${BINARY_NAME}"
            rm -rf "${temp_dir}"
            exit 1
        fi
    fi
    
    log_info "Extracting archive..."
    tar -xzf "${archive_file}" -C "${temp_dir}"
    
    if [ ! -f "${temp_dir}/${BINARY_NAME}" ]; then
        log_error "Binary not found in archive"
        rm -rf "${temp_dir}"
        exit 1
    fi
    
    echo "${temp_dir}/${BINARY_NAME}"
}

# Install binary
install_binary() {
    local binary_path="$1"
    local install_path=""
    
    # Try system-wide installation first
    if [ -w "${INSTALL_DIR}" ] || [ "$(id -u)" -eq 0 ]; then
        install_path="${INSTALL_DIR}/${BINARY_NAME}"
        log_info "Installing to ${install_path} (system-wide)"
    else
        # Fall back to user installation
        mkdir -p "${USER_INSTALL_DIR}"
        install_path="${USER_INSTALL_DIR}/${BINARY_NAME}"
        log_info "Installing to ${install_path} (user directory)"
        log_warning "Make sure ${USER_INSTALL_DIR} is in your PATH"
    fi
    
    if cp "${binary_path}" "${install_path}"; then
        chmod +x "${install_path}"
        log_success "Successfully installed ${BINARY_NAME} to ${install_path}"
    else
        log_error "Failed to install ${BINARY_NAME}"
        exit 1
    fi
    
    echo "${install_path}"
}

# Verify installation
verify_installation() {
    local install_path="$1"
    
    log_info "Verifying installation..."
    
    if "${install_path}" --version >/dev/null 2>&1; then
        local version_output=$("${install_path}" --version 2>/dev/null)
        log_success "Installation verified successfully!"
        log_info "Version: ${version_output}"
    else
        log_warning "Installation completed but verification failed"
        log_info "You may need to add the installation directory to your PATH"
    fi
}

# Show usage instructions
show_usage() {
    echo
    log_info "Usage instructions:"
    echo "  ${BINARY_NAME} --help                    # Show help"
    echo "  ${BINARY_NAME} --dry-run               # Preview changes"
    echo "  ${BINARY_NAME} --verbose               # Enable verbose output"
    echo "  ${BINARY_NAME} restore                 # Restore from backup"
    echo
    log_info "For more information, visit: https://github.com/${REPO}"
}

# Main installation process
main() {
    log_info "kubectx-manager Installation Script"
    log_info "Repository: https://github.com/${REPO}"
    echo
    
    # Detect platform
    local platform=$(detect_platform)
    log_info "Detected platform: ${platform}"
    
    # Get latest version
    local version=$(get_latest_version)
    if [ -z "${version}" ]; then
        log_error "Could not determine latest version"
        exit 1
    fi
    log_info "Latest version: ${version}"
    
    # Download binary
    local binary_path=$(download_binary "${version}" "${platform}")
    
    # Install binary
    local install_path=$(install_binary "${binary_path}")
    
    # Clean up
    rm -rf "$(dirname "${binary_path}")"
    
    # Verify installation
    verify_installation "${install_path}"
    
    # Show usage
    show_usage
    
    log_success "Installation completed successfully!"
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "kubectx-manager Installation Script"
        echo
        echo "This script downloads and installs the latest version of kubectx-manager"
        echo "from GitHub releases."
        echo
        echo "Usage: $0 [OPTIONS]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --version, -v  Show script version"
        echo
        echo "The script will:"
        echo "  1. Detect your platform (OS and architecture)"
        echo "  2. Download the latest release from GitHub"
        echo "  3. Install to /usr/local/bin (or ~/bin if no permissions)"
        echo "  4. Verify the installation"
        echo
        exit 0
        ;;
    --version|-v)
        echo "kubectx-manager installation script v1.0.0"
        exit 0
        ;;
    "")
        main
        ;;
    *)
        log_error "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
esac
