#!/usr/bin/env bash

# PostgreSQL + TimescaleDB Quick Setup Script
# Starts TimescaleDB with Docker Compose for development

set -euo pipefail

# Script directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.timescale.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi

    print_success "Docker is running"
}

# Check if docker-compose is available
check_docker_compose() {
    if command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker-compose"
    elif docker compose version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker compose"
    else
        print_error "docker-compose is not available. Please install docker-compose."
        exit 1
    fi
    
    print_success "Docker Compose is available: $DOCKER_COMPOSE_CMD"
}

# Load environment variables
load_environment() {
    # Load .env if present
    if [ -f "$REPO_ROOT/.env" ]; then
        # shellcheck disable=SC1090
        set -a
        . "$REPO_ROOT/.env"
        set +a
        print_success ".env file loaded"
    else
        print_warning ".env file not found, using defaults"
    fi

    # Provide sensible defaults if not set
    : "${POSTGRES_USER:=monitoring}"
    : "${POSTGRES_PASSWORD:=monitoring}"
    : "${POSTGRES_DB:=monitoring}"
    export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
    export COMPOSE_PROJECT_NAME=monitoring
}

# Start TimescaleDB
start_timescaledb() {
    print_status "Starting TimescaleDB with Docker Compose..."
    
    if [ ! -f "$COMPOSE_FILE" ]; then
        print_error "docker-compose.timescale.yml not found at: $COMPOSE_FILE"
        exit 1
    fi

    # Start TimescaleDB
    $DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" up -d

    print_success "TimescaleDB container started"
}

# Wait for database to be ready
wait_for_database() {
    print_status "Waiting for database to be ready..."
    
    local max_attempts=60
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if $DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" exec -T timescaledb pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" &> /dev/null; then
            print_success "Database is ready!"
            return 0
        fi
        
        printf "."
        sleep 1
        ((attempt++))
    done
    
    echo ""
    print_error "Database failed to start after ${max_attempts} seconds"
    return 1
}

# Test database connection
test_connection() {
    print_status "Testing database connection..."

    # Pass password via env to avoid interactive prompt
    # Capture error output so we can surface real failure reasons (e.g., permissions)
    local output
    if output=$($DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" \
        timescaledb psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT version();" 2>&1); then
        print_success "Database connection successful"
        
        # Check TimescaleDB extension
        if $DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" timescaledb psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT * FROM pg_extension WHERE extname = 'timescaledb';" | grep -q timescaledb; then
            print_success "TimescaleDB extension is available"
        else
            print_warning "TimescaleDB extension may not be properly installed"
        fi
    else
        print_error "Failed to connect to database"
        echo "$output"
        return 1
    fi
}

# Show connection information
show_connection_info() {
    echo ""
    print_status "📋 Database connection information:"
    echo "  Host: localhost"
    echo "  Port: 5432"
    echo "  Database: $POSTGRES_DB"
    echo "  Username: $POSTGRES_USER"
    echo "  Password: $POSTGRES_PASSWORD"
    echo ""
    print_status "🔗 To connect manually:"
    echo "  $DOCKER_COMPOSE_CMD -f \"$COMPOSE_FILE\" exec -e PGPASSWORD=\"$POSTGRES_PASSWORD\" timescaledb psql -U $POSTGRES_USER -d $POSTGRES_DB"
    echo ""
    print_status "⚙️  Your .env file should contain:"
    echo "  POSTGRES_HOST=localhost"
    echo "  POSTGRES_PORT=5432"
    echo "  POSTGRES_DB=$POSTGRES_DB"
    echo "  POSTGRES_USER=$POSTGRES_USER"
    echo "  POSTGRES_PASSWORD=$POSTGRES_PASSWORD"
}

# Show management commands
show_management_info() {
    echo ""
    print_status "🛠️  Management commands:"
    echo "  Stop:      $0 --stop"
    echo "  Restart:   $0 --restart" 
    echo "  Logs:      $0 --logs"
    echo "  Clean:     $0 --clean (removes all data)"
    echo ""
    print_status "📖 For more help:"
    echo "  Documentation: docs/postgresql-setup.md"
    echo "  Troubleshooting: docs/troubleshooting.md"
}

# Stop TimescaleDB
stop_timescaledb() {
    print_status "Stopping TimescaleDB..."
    $DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" down
    print_success "TimescaleDB stopped"
}

# Restart TimescaleDB
restart_timescaledb() {
    print_status "Restarting TimescaleDB..."
    $DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" restart
    print_success "TimescaleDB restarted"
}

# Show logs
show_logs() {
    print_status "Showing TimescaleDB logs (Ctrl+C to exit)..."
    $DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" logs -f
}

# Clean (remove all data)
clean_timescaledb() {
    print_warning "⚠️  This will remove ALL data in the database!"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        $DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" down -v
        print_success "TimescaleDB stopped and all data removed"
    else
        print_status "Operation cancelled"
    fi
}

# Show help
show_help() {
    echo "PostgreSQL + TimescaleDB Quick Setup"
    echo ""
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Options:"
    echo "  (no args)    Start TimescaleDB and show connection info"
    echo "  --stop       Stop TimescaleDB container"
    echo "  --restart    Restart TimescaleDB container"
    echo "  --logs       Show TimescaleDB logs"
    echo "  --clean      Stop and remove all data (destructive!)"
    echo "  --help, -h   Show this help"
    echo ""
    echo "Documentation:"
    echo "  📖 Setup Guide: docs/postgresql-setup.md"
    echo "  🚀 Production:  docs/production-deployment.md"
    echo "  🔍 Troubleshoot: docs/troubleshooting.md"
}

# Main execution
main() {
    print_status "🐳 Starting PostgreSQL + TimescaleDB setup..."
    
    check_docker
    check_docker_compose
    load_environment
    start_timescaledb
    
    if wait_for_database && test_connection; then
        print_success "🎉 TimescaleDB is running and ready!"
        show_connection_info
        show_management_info
    else
        print_error "❌ Setup failed. Check the logs with:"
        echo "  $0 --logs"
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
    --stop)
        load_environment
        check_docker_compose
        stop_timescaledb
        ;;
    --restart)
        load_environment
        check_docker_compose
        restart_timescaledb
        ;;
    --logs)
        load_environment
        check_docker_compose
        show_logs
        ;;
    --clean)
        load_environment
        check_docker_compose
        clean_timescaledb
        ;;
    --help|-h)
        show_help
        ;;
    "")
        main
        ;;
    *)
        print_error "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
esac
