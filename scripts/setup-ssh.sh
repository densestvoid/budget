#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# SSH key setup for DigitalOcean deployment
setup_ssh_keys() {
    print_status "Setting up SSH keys for DigitalOcean deployment..."
    
    SSH_DIR="$HOME/.ssh"
    PRIVATE_KEY="$SSH_DIR/id_rsa"
    PUBLIC_KEY="$SSH_DIR/id_rsa.pub"
    
    # Create SSH directory if it doesn't exist
    mkdir -p "$SSH_DIR"
    chmod 700 "$SSH_DIR"
    
    # Check if SSH keys already exist
    if [[ -f "$PRIVATE_KEY" && -f "$PUBLIC_KEY" ]]; then
        print_warning "SSH keys already exist at $PRIVATE_KEY and $PUBLIC_KEY"
        echo "Do you want to use existing keys? (y/N)"
        read -r response
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            print_status "Generating new SSH keys..."
            ssh-keygen -t rsa -b 4096 -f "$PRIVATE_KEY" -N "" -C "budget-app-deployment"
        fi
    else
        print_status "Generating new SSH keys..."
        ssh-keygen -t rsa -b 4096 -f "$PRIVATE_KEY" -N "" -C "budget-app-deployment"
    fi
    
    # Set proper permissions
    chmod 600 "$PRIVATE_KEY"
    chmod 644 "$PUBLIC_KEY"
    
    print_success "SSH keys are ready"
    print_status "Public key location: $PUBLIC_KEY"
    print_status "Private key location: $PRIVATE_KEY"
    
    echo ""
    print_status "Your public key:"
    cat "$PUBLIC_KEY"
    echo ""
    
    print_warning "Make sure to add this public key to your DigitalOcean account if you haven't already."
}

main() {
    setup_ssh_keys
}

main "$@"