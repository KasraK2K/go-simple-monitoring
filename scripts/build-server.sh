#!/bin/bash

# Build the monitoring server
echo "Building monitoring server..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Detect current platform
get_current_platform() {
    case "$(uname -s)" in
        Darwin) echo "mac" ;;
        Linux) echo "linux" ;;
        MINGW*|CYGWIN*|MSYS*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

# Build for specific OS
build_for_os() {
    local target_os=$1
    local current_os=$(get_current_platform)
    local output_name="monitoring-${target_os}"
    local cgo_enabled=0
    
    # Enable CGO only for native builds (same platform)
    if [ "$target_os" = "$current_os" ]; then
        cgo_enabled=1
        echo "🔧 Native build detected - CGO enabled for SQLite support"
    else
        echo "⚠️  Cross-compiling - CGO disabled (use file storage)"
    fi
    
    case $target_os in
        "windows")
            echo "🔨 Building for Windows..."
            GOOS=windows GOARCH=amd64 CGO_ENABLED=$cgo_enabled go build -o "bin/${output_name}.exe" ./cmd/
            ;;
        "linux")
            echo "🔨 Building for Linux..."
            GOOS=linux GOARCH=amd64 CGO_ENABLED=$cgo_enabled go build -o "bin/${output_name}" ./cmd/
            ;;
        "mac")
            echo "🔨 Building for macOS..."
            GOOS=darwin GOARCH=amd64 CGO_ENABLED=$cgo_enabled go build -o "bin/${output_name}" ./cmd/
            ;;
        *)
            echo "❌ Unsupported OS: $target_os"
            echo "Supported: windows, linux, mac"
            return 1
            ;;
    esac
    
    if [ $? -eq 0 ]; then
        echo "✅ Built: ./bin/${output_name}"
        if [ $cgo_enabled -eq 1 ]; then
            echo "   📊 SQLite database support: YES"
        else
            echo "   📊 SQLite database support: NO (use 'storage': 'file' in configs.json)"
        fi
    else
        echo "❌ Build failed for $target_os"
        return 1
    fi
}

# Validate arguments
if [ $# -eq 0 ]; then
    echo "❌ No OS specified!"
    echo "Usage: $0 [windows|linux|mac] [additional OS...]"
    echo "Example: $0 windows linux mac"
    exit 1
fi

# Build for each specified OS
for os in "$@"; do
    build_for_os "$os" || exit 1
done

echo ""
echo "🚀 Server binaries built successfully!"
echo "Note: CGO disabled for cross-platform compatibility"