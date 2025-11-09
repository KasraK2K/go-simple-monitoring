#!/bin/bash

# Build monitoring server for Linux using persistent Docker container
echo "🐳 Building monitoring server for Linux using Docker..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

CONTAINER_NAME="build-go-linux"

# Function to create the build container
create_container() {
    echo "📦 Creating persistent build container: $CONTAINER_NAME"
    docker run -d \
        --platform linux/amd64 \
        --name $CONTAINER_NAME \
        -v "$PWD":/workspace \
        -w /workspace \
        golang:1.24-bullseye \
        tail -f /dev/null
    
    # Install build dependencies once
    echo "⚙️  Installing build dependencies..."
    docker exec $CONTAINER_NAME apt-get update && docker exec $CONTAINER_NAME apt-get install -y gcc libc6-dev libsqlite3-dev
    echo "✅ Build container ready"
}

# Check if container exists and is running
if ! docker ps -q -f name=$CONTAINER_NAME | grep -q .; then
    if docker ps -aq -f name=$CONTAINER_NAME | grep -q .; then
        echo "🔄 Starting existing container: $CONTAINER_NAME"
        docker start $CONTAINER_NAME
    else
        create_container
    fi
fi

echo "🔨 Building for Linux with CGO and SQLite support..."

# Build using the persistent container
docker exec $CONTAINER_NAME sh -c "
    # Set Go proxy for better reliability
    export GOPROXY=https://proxy.golang.org,direct
    export GOSUMDB=sum.golang.org
    
    # Clean module cache and retry if needed
    echo 'Downloading dependencies...'
    go mod download || (go clean -modcache && go mod download)
    
    echo 'Generating templ files...'
    go install github.com/a-h/templ/cmd/templ@latest
    templ generate ./web/views
    
    echo 'Building binary...'
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/monitoring-linux ./cmd/
"

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "✅ Built: ./bin/monitoring-linux"
    echo "   📊 SQLite database support: YES"
    echo "   🐳 Built in persistent Docker container"
    echo "   ⚡ Next builds will be faster (container reused)"
    echo ""
    echo "🚀 Deploy to your server:"
    echo "   scp bin/monitoring-linux user@server:/opt/monitoring/monitoring"
    echo "   ssh user@server 'sudo chown monitoring:monitoring /opt/monitoring/monitoring'"
    echo "   ssh user@server 'sudo systemctl restart monitoring'"
    echo ""
    echo "💡 To remove build container: docker rm -f $CONTAINER_NAME"
else
    echo "❌ Docker build failed"
    exit 1
fi