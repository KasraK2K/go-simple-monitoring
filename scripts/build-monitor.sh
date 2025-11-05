#!/bin/bash

# Build the monitoring CLI tool
echo "Building monitoring CLI tool..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Function to build for specific OS
build_for_os() {
    local os=$1
    local arch="amd64"
    local ext=""
    local output_name="monitor"
    
    case $os in
        "windows")
            export GOOS=windows
            export GOARCH=amd64
            ext=".exe"
            output_name="monitor-windows"
            ;;
        "linux")
            export GOOS=linux
            export GOARCH=amd64
            output_name="monitor-linux"
            ;;
        "mac")
            export GOOS=darwin
            export GOARCH=amd64
            output_name="monitor-mac"
            ;;
        *)
            echo "❌ Unsupported OS: $os"
            echo "Supported OS: windows, linux, mac"
            return 1
            ;;
    esac
    
    echo "🔨 Building for $os..."
    go build -o "bin/${output_name}${ext}" ./cmd/monitor/
    
    if [ $? -eq 0 ]; then
        echo "✅ Built for $os: ./bin/${output_name}${ext}"
    else
        echo "❌ Failed to build for $os"
        return 1
    fi
}

# Check if arguments are provided
if [ $# -eq 0 ]; then
    echo "❌ No OS specified!"
    echo "Usage: $0 [windows|linux|mac] [additional OS...]"
    echo "Example: $0 windows linux mac"
    exit 1
fi

# Build for each specified OS
for os in "$@"; do
    build_for_os "$os"
    if [ $? -ne 0 ]; then
        exit 1
    fi
done

echo ""
echo "🚀 Usage examples:"
echo "  Local monitoring:     ./bin/monitor-[os]"
echo "  Remote monitoring:    ./bin/monitor-[os] -url http://localhost:3500/monitoring"
echo "  With authentication:  ./bin/monitor-[os] -url http://localhost:3500/monitoring -token YOUR_TOKEN"
echo "  Compact mode:         ./bin/monitor-[os] -compact"
echo "  Show details:         ./bin/monitor-[os] -details"
echo "  Custom refresh rate:  ./bin/monitor-[os] -refresh 1s"