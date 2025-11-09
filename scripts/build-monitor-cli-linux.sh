#!/bin/bash
set -euo pipefail

# Build monitor CLI for Linux using persistent Docker container
echo "🐳 Building monitor CLI for Linux using Docker..."

mkdir -p bin

if ! command -v docker >/dev/null 2>&1; then
    echo "❌ Docker is not installed or not in PATH."
    exit 1
fi

if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

CONTAINER_NAME="build-go-linux"

create_container() {
    echo "📦 Creating persistent build container: $CONTAINER_NAME"
    docker run -d \
        --platform linux/amd64 \
        --name "$CONTAINER_NAME" \
        -v "$PWD":/workspace \
        -w /workspace \
        golang:1.24-bullseye \
        tail -f /dev/null

    echo "⚙️  Installing build dependencies..."
    docker exec "$CONTAINER_NAME" apt-get update
    docker exec "$CONTAINER_NAME" apt-get install -y gcc libc6-dev
    echo "✅ Build container ready"
}

if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
    if docker ps -aq -f name="$CONTAINER_NAME" | grep -q .; then
        echo "🔄 Starting existing container: $CONTAINER_NAME"
        docker start "$CONTAINER_NAME"
    else
        create_container
    fi
fi

echo "🔨 Building CLI for Linux with CGO support..."

docker exec "$CONTAINER_NAME" sh -c "
    set -e
    export GOPROXY=https://proxy.golang.org,direct
    export GOSUMDB=sum.golang.org
    echo '📦 Downloading dependencies...'
    go mod download || (go clean -modcache && go mod download)
    echo '🚧 Building monitor CLI binary...'
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/monitor-cli-linux ./cmd/monitor
"

if [ $? -eq 0 ]; then
    echo "✅ Built: ./bin/monitor-cli-linux"
    echo "   🐳 Built in persistent Docker container ($CONTAINER_NAME)"
    echo "   ⚡ Next builds will be faster (container reused)"
    echo ""
    echo "🚀 Deploy to your server:"
    echo "   scp bin/monitor-cli-linux user@server:/tmp/monitor-cli"
    echo "   ssh user@server 'sudo mv /tmp/monitor-cli /usr/local/bin/monitor-cli'"
    echo "   ssh user@server 'sudo chmod +x /usr/local/bin/monitor-cli'"
    echo ""
    echo "💡 Remove build container when done: docker rm -f $CONTAINER_NAME"
else
    echo "❌ Docker build failed"
    exit 1
fi
