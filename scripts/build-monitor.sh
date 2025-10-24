#!/bin/bash

# Build the monitoring CLI tool
echo "Building monitoring CLI tool..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build for current platform
go build -o bin/monitor ./cmd/monitor/

echo "✅ Monitor CLI built successfully!"
echo "📍 Location: ./bin/monitor"
echo ""
echo "🚀 Usage examples:"
echo "  Local monitoring:     ./bin/monitor"
echo "  Remote monitoring:    ./bin/monitor -url http://localhost:3500/monitoring"
echo "  With authentication:  ./bin/monitor -url http://localhost:3500/monitoring -token YOUR_TOKEN"
echo "  Compact mode:         ./bin/monitor -compact"
echo "  Show details:         ./bin/monitor -details"
echo "  Custom refresh rate:  ./bin/monitor -refresh 1s"