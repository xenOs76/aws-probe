#!/usr/bin/env sh
# aws-probe install script
# Installs the latest version of aws-probe for the current architecture

REPO="xenos76/aws-probe"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="aws-probe"

# Check dependencies
check_deps() {
    for cmd in curl tar; do
        if ! command -v "$cmd" > /dev/null 2>&1; then
            echo "Error: $cmd is required but not installed" >&2
            return 1
        fi
    done
    return 0
}

# Install function
install() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) BIN_ARCH="Linux_amd64" ;;
        aarch64|arm64) BIN_ARCH="Linux_arm64" ;;
        *) echo "Unsupported architecture: $ARCH" >&2; return 1 ;;
    esac

    echo "Detected architecture: $ARCH (binary: $BIN_ARCH)"

    # Get the latest release download URL
    DOWNLOAD_URL=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep "browser_download_url.*${BIN_ARCH}\.tar\.gz" \
        | cut -d '"' -f 4)

    if [ -z "$DOWNLOAD_URL" ]; then
        echo "No binary found for architecture: $BIN_ARCH" >&2
        return 1
    fi

    echo "Downloading from: $DOWNLOAD_URL"

    # Download and extract
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR" || return 1

    curl -sSL "$DOWNLOAD_URL" -o "${BINARY_NAME}.tar.gz" || return 1
    tar -xzf "${BINARY_NAME}.tar.gz" || return 1

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        cp "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
        echo "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
    else
        echo "Cannot write to $INSTALL_DIR (need sudo?)" >&2
        echo "Run with sudo or set INSTALL_DIR environment variable" >&2
        cd /
        rm -rf "$TEMP_DIR"
        return 1
    fi

    # Cleanup
    cd / || return 1
    rm -rf "$TEMP_DIR"

    # Verify
    VERSION=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>&1 | awk '{print $NF}')
    echo "Installed version: $VERSION"
}

# Main
if ! check_deps; then
    exit 1
fi

install
exit $?