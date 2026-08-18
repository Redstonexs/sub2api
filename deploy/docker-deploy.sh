#!/bin/bash
# =============================================================================
# Sub2API Docker Deployment Preparation Script
# =============================================================================
# This script prepares deployment files for Sub2API:
#   - Downloads docker-compose.local.yml and .env.example
#   - Generates secure secrets (ADMIN_PASSWORD, JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD)
#   - Creates necessary data directories
#
# After running this script, you can start services with:
#   docker-compose up -d
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub raw content base URL
GITHUB_RAW_URL="https://raw.githubusercontent.com/Redstonexs/sub2api/main/deploy"

# Print colored message
print_info() {
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

# Generate random secret
generate_secret() {
    openssl rand -hex 32
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Main installation function
main() {
    echo ""
    echo "=========================================="
    echo "  Sub2API Deployment Preparation"
    echo "=========================================="
    echo ""

    # Check if openssl is available
    if ! command_exists openssl; then
        print_error "openssl is not installed. Please install openssl first."
        exit 1
    fi

    # Check if deployment already exists
    if [ -f "docker-compose.yml" ] && [ -f ".env" ]; then
        print_warning "Deployment files already exist in current directory."
        read -p "Overwrite existing files? (y/N): " -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Cancelled."
            exit 0
        fi
    fi

    # Download docker-compose.local.yml and save as docker-compose.yml
    print_info "Downloading docker-compose.yml..."
    if command_exists curl; then
        curl -sSL "${GITHUB_RAW_URL}/docker-compose.local.yml" -o docker-compose.yml
    elif command_exists wget; then
        wget -q "${GITHUB_RAW_URL}/docker-compose.local.yml" -O docker-compose.yml
    else
        print_error "Neither curl nor wget is installed. Please install one of them."
        exit 1
    fi
    print_success "Downloaded docker-compose.yml"

    print_info "Downloading full-instance migration runner..."
    if command_exists curl; then
        curl -sSL "${GITHUB_RAW_URL}/full-instance-migration.sh" -o full-instance-migration.sh
    else
        wget -q "${GITHUB_RAW_URL}/full-instance-migration.sh" -O full-instance-migration.sh
    fi
    print_success "Downloaded full-instance migration runner"

    # Download .env.example
    print_info "Downloading .env.example..."
    if command_exists curl; then
        curl -sSL "${GITHUB_RAW_URL}/.env.example" -o .env.example
    else
        wget -q "${GITHUB_RAW_URL}/.env.example" -O .env.example
    fi
    print_success "Downloaded .env.example"

    # Generate .env file with auto-generated secrets
    print_info "Generating secure secrets..."
    echo ""

    # Generate secrets
    JWT_SECRET=$(generate_secret)
    TOTP_ENCRYPTION_KEY=$(generate_secret)
    POSTGRES_PASSWORD=$(generate_secret)
    ADMIN_PASSWORD=$(generate_secret)

    # Create .env from .env.example with restricted permissions
    # umask 077 prevents other users from reading the file during creation
    umask 077
    cp .env.example .env
    chmod 600 .env

    # Update .env with generated secrets using a temp sed script file.
    # This avoids exposing secrets in the process list (visible via ps).
    local sed_script
    sed_script=$(mktemp)
    chmod 600 "$sed_script"
    cat > "$sed_script" <<-SEDEOF
	s/^JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/
	s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=$TOTP_ENCRYPTION_KEY/
	s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$POSTGRES_PASSWORD/
	s/^ADMIN_PASSWORD=.*/ADMIN_PASSWORD=$ADMIN_PASSWORD/
	SEDEOF

    if sed --version >/dev/null 2>&1; then
        # GNU sed (Linux)
        sed -i -f "$sed_script" .env
    else
        # BSD sed (macOS)
        sed -i '' -f "$sed_script" .env
    fi
    rm -f "$sed_script"

    # Create data directories
    print_info "Creating data directories..."
    mkdir -p data postgres_data redis_data migration_artifacts
    print_success "Created data directories"

    # Set secure permissions for .env file (readable/writable only by owner)
    chmod 600 .env
    echo ""

    # Display completion message
    echo "=========================================="
    echo "  Preparation Complete!"
    echo "=========================================="
    echo ""
    print_warning "Secure credentials have been saved to the mode-600 .env file."
    print_warning "Do not print, commit, or share that file."
    echo ""
    echo "Directory structure:"
    echo "  docker-compose.yml        - Docker Compose configuration"
    echo "  full-instance-migration.sh - Compose migration runner"
    echo "  .env                      - Environment variables (generated secrets)"
    echo "  .env.example              - Example template (for reference)"
    echo "  data/                     - Application data (will be created on first run)"
    echo "  postgres_data/            - PostgreSQL data"
    echo "  redis_data/               - Redis data"
    echo "  migration_artifacts/      - Full-instance migration archives"
    echo ""
    echo "Next steps:"
    echo "  1. (Optional) Edit .env to customize configuration"
    echo "  2. Start services:"
    echo "     docker-compose up -d"
    echo ""
    echo "  3. View logs:"
    echo "     docker-compose logs -f sub2api"
    echo ""
    echo "  4. Access Web UI:"
    echo "     http://localhost:8080"
    echo ""
}

# Run main function
main "$@"
