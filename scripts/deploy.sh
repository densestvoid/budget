#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TERRAFORM_DIR="terraform"
BUILD_DIR="build"
DOCKER_IMAGE="budget-app"

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

check_requirements() {
    print_status "Checking requirements..."
    
    # Check if terraform is installed
    if ! command -v terraform &> /dev/null; then
        print_error "Terraform is not installed. Please install Terraform first."
        exit 1
    fi
    
    # Check if docker is installed
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check if terraform.tfvars exists
    if [ ! -f "$TERRAFORM_DIR/terraform.tfvars" ]; then
        print_error "terraform.tfvars not found. Please copy terraform.tfvars.example to terraform.tfvars and configure it."
        exit 1
    fi
    
    print_success "All requirements met"
}

build_application() {
    print_status "Building application..."
    
    # Create build directory
    mkdir -p $BUILD_DIR
    
    # Build Go application
    print_status "Building Go binary..."
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $BUILD_DIR/budget main.go
    
    # Copy necessary files
    cp Dockerfile $BUILD_DIR/
    cp -r templates $BUILD_DIR/ 2>/dev/null || true
    cp config.yaml $BUILD_DIR/ 2>/dev/null || true
    
    print_success "Application built successfully"
}

build_docker_image() {
    print_status "Building Docker image..."
    
    cd $BUILD_DIR
    docker build -t $DOCKER_IMAGE:latest .
    cd ..
    
    print_success "Docker image built successfully"
}

deploy_infrastructure() {
    print_status "Deploying infrastructure with Terraform..."
    
    cd $TERRAFORM_DIR
    
    # Initialize Terraform
    print_status "Initializing Terraform..."
    terraform init
    
    # Plan deployment
    print_status "Planning Terraform deployment..."
    terraform plan
    
    # Apply deployment
    print_status "Applying Terraform deployment..."
    terraform apply -auto-approve
    
    cd ..
    
    print_success "Infrastructure deployed successfully"
}

get_deployment_info() {
    print_status "Getting deployment information..."
    
    cd $TERRAFORM_DIR
    
    APP_IP=$(terraform output -raw app_ip_address)
    APP_URL=$(terraform output -raw application_url)
    DB_TYPE=$(terraform output -raw database_type)
    COST=$(terraform output -raw estimated_monthly_cost)
    AUTO_TERM=$(terraform output -raw auto_termination_info)
    
    cd ..
    
    print_success "Deployment information retrieved"
    echo -e "${GREEN}Application IP:${NC} $APP_IP"
    echo -e "${GREEN}Application URL:${NC} $APP_URL"
    echo -e "${GREEN}Database Type:${NC} $DB_TYPE"
    echo -e "${GREEN}Estimated Cost:${NC} $COST"
    echo -e "${YELLOW}Auto-termination:${NC} $AUTO_TERM"
}

deploy_application() {
    print_status "Deploying application to server..."
    
    cd $TERRAFORM_DIR
    APP_IP=$(terraform output -raw app_ip_address)
    cd ..
    
    # Save Docker image
    print_status "Saving Docker image..."
    docker save $DOCKER_IMAGE:latest | gzip > $BUILD_DIR/budget-app.tar.gz
    
    # Copy image to server
    print_status "Copying Docker image to server..."
    scp -o StrictHostKeyChecking=no $BUILD_DIR/budget-app.tar.gz root@$APP_IP:/tmp/
    
    # Load and run image on server
    print_status "Loading and running application on server..."
    ssh -o StrictHostKeyChecking=no root@$APP_IP << 'EOF'
        # Load Docker image
        docker load < /tmp/budget-app.tar.gz
        
        # Run deployment script
        if [ -f /home/budget/deploy.sh ]; then
            su - budget -c "cd /home/budget && ./deploy.sh"
        else
            echo "Deploy script not found, running containers manually..."
            su - budget -c "cd /home/budget && docker-compose up -d"
        fi
        
        # Clean up
        rm /tmp/budget-app.tar.gz
EOF
    
    print_success "Application deployed successfully"
}

run_migrations() {
    print_status "Running database migrations..."
    
    cd $TERRAFORM_DIR
    APP_IP=$(terraform output -raw app_ip_address)
    cd ..
    
    # Run migrations on the server
    ssh -o StrictHostKeyChecking=no root@$APP_IP << 'EOF'
        # Wait for PostgreSQL to be ready
        su - budget -c "
            cd /home/budget
            echo 'Waiting for PostgreSQL to be ready...'
            timeout 60 bash -c 'until docker-compose exec -T postgres pg_isready -U budget_user -d budget; do sleep 2; done'
            
            # Run migrations using the deployed application
            echo 'Running database migrations...'
            docker-compose exec -T app ./budget migrate || echo 'Migration command not available, skipping...'
        "
EOF
    
    print_success "Database migrations completed"
}

cleanup() {
    print_status "Cleaning up build artifacts..."
    rm -rf $BUILD_DIR
    print_success "Cleanup completed"
}

main() {
    print_status "Starting Budget App deployment..."
    
    case "${1:-all}" in
        "check")
            check_requirements
            ;;
        "build")
            check_requirements
            build_application
            build_docker_image
            ;;
        "infrastructure")
            check_requirements
            deploy_infrastructure
            ;;
        "deploy")
            check_requirements
            build_application
            build_docker_image
            deploy_application
            run_migrations
            cleanup
            ;;
        "info")
            get_deployment_info
            ;;
        "destroy")
            print_warning "This will destroy all infrastructure. Are you sure? (y/N)"
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                cd $TERRAFORM_DIR
                terraform destroy
                cd ..
                print_success "Infrastructure destroyed"
            else
                print_status "Destruction cancelled"
            fi
            ;;
        "all"|*)
            check_requirements
            build_application
            build_docker_image
            deploy_infrastructure
            get_deployment_info
            deploy_application
            run_migrations
            cleanup
            print_success "Full deployment completed!"
            get_deployment_info
            echo -e "${GREEN}Your Budget App is now running at:${NC} $APP_URL"
            echo -e "${YELLOW}⚠️  Note: This deployment will auto-terminate after 30 minutes to save costs${NC}"
            ;;
    esac
}

# Show usage if help is requested
if [[ "${1}" == "help" || "${1}" == "-h" || "${1}" == "--help" ]]; then
    echo "Budget App Deployment Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  all            Full deployment (default)"
    echo "  check          Check requirements"
    echo "  build          Build application and Docker image"
    echo "  infrastructure Deploy infrastructure only"
    echo "  deploy         Deploy application only (requires infrastructure)"
    echo "  info           Show deployment information"
    echo "  destroy        Destroy all infrastructure"
    echo "  help           Show this help message"
    echo ""
    exit 0
fi

main "$1"