#!/bin/bash

# Docker Monitor Deployment Script

set -e

echo "🐳 Docker Container Monitor Deployment"
echo "======================================="

# Check if Docker and Docker Compose are installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Create logs directory
mkdir -p logs
echo "📁 Created logs directory"

# Copy .env.example to .env if it doesn't exist
if [ ! -f .env ]; then
    cp .env.example .env
    echo "📝 Created .env file from template. Please configure it as needed."
fi

# Ask user which configuration to use
echo ""
echo "Select deployment configuration:"
echo "1) Console logging only (default)"
echo "2) Console + CSV + Excel logging"
echo "3) CloudWatch logging (requires AWS credentials)"
echo "4) Use custom config.json file"
echo ""
read -p "Enter choice (1-4): " choice

case $choice in
    1)
        echo "🚀 Deploying with console logging..."
        docker-compose up -d docker-monitor
        ;;
    2)
        echo "🚀 Deploying with multiple local drivers..."
        docker-compose --profile multi-logger up -d docker-monitor-multi
        ;;
    3)
        echo "🚀 Deploying with CloudWatch logging..."
        if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
            echo "⚠️  Please set AWS credentials in .env file first:"
            echo "   AWS_ACCESS_KEY_ID=your_key"
            echo "   AWS_SECRET_ACCESS_KEY=your_secret"
            echo "   AWS_REGION=your_region"
            exit 1
        fi
        docker-compose --profile cloudwatch up -d docker-monitor-cloudwatch
        ;;
    4)
        echo "🚀 Deploying with config.json file..."
        if [ ! -f config.json ]; then
            echo "❌ config.json file not found. Please create it first."
            exit 1
        fi
        docker-compose up -d docker-monitor
        ;;
    *)
        echo "🚀 Deploying with default console logging..."
        docker-compose up -d docker-monitor
        ;;
esac

echo ""
echo "✅ Deployment completed!"
echo ""
echo "📊 Monitor status:"
docker-compose ps

echo ""
echo "📋 Useful commands:"
echo "   View logs:           docker-compose logs -f docker-monitor"
echo "   Stop monitor:        docker-compose down"
echo "   Restart monitor:     docker-compose restart docker-monitor"
echo "   View CSV files:      ls -la logs/"
echo ""

# Wait a moment and show initial logs
echo "🔍 Initial logs (first 20 lines):"
sleep 3
docker-compose logs --tail=20 docker-monitor || true