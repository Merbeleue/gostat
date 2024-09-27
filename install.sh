#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status.

ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

DOWNLOAD_URL="https://github.com/Merbeleue/gostat/releases/latest/download/gostat_Linux_${ARCH}.tar.gz"
TEMP_DIR=$(mktemp -d)
TEMP_FILE="${TEMP_DIR}/gostat.tar.gz"

echo "Downloading from: $DOWNLOAD_URL"
if ! curl -L "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
    echo "Failed to download the file"
    exit 1
fi

echo "Download completed. File size: $(du -h "$TEMP_FILE" | cut -f1)"

echo "Extracting files..."
if ! tar -xzvf "$TEMP_FILE" -C "$TEMP_DIR"; then
    echo "Failed to extract the archive"
    exit 1
fi

echo "Extracted files:"
ls -l "$TEMP_DIR"

if [ ! -f "${TEMP_DIR}/gostat" ]; then
    echo "gostat binary not found in the extracted files"
    exit 1
fi

echo "Moving gostat to /usr/local/bin/"
if ! sudo mv "${TEMP_DIR}/gostat" /usr/local/bin/; then
    echo "Failed to move gostat to /usr/local/bin/"
    exit 1
fi

rm -rf "$TEMP_DIR"

echo "gostat has been installed to /usr/local/bin/gostat"
echo "You can now run it by typing 'gostat' in your terminal."

# Verify installation
if command -v gostat &> /dev/null; then
    echo "Verification: gostat is successfully installed and accessible."
    gostat --version
else
    echo "Verification failed: gostat command not found in PATH"
    exit 1
fi