#!/bin/bash

# vman project development environment setup script

set -e

echo "Setting up vman development environment..."

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
REQUIRED_VERSION="1.21"

if ! printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo "❌ Go version $GO_VERSION is too old. Please upgrade to Go $REQUIRED_VERSION or later."
    exit 1
fi

echo "✅ Go version $GO_VERSION is compatible"

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download

# Verify dependencies
echo "🔍 Verifying dependencies..."
go mod verify

# Install development tools
echo "🔧 Installing development tools..."

# golangci-lint for linting
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
else
    echo "✅ golangci-lint already installed"
fi

# goimports for code formatting
if ! command -v goimports &> /dev/null; then
    echo "Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@latest
else
    echo "✅ goimports already installed"
fi

# Create necessary directories
echo "📁 Creating necessary directories..."
mkdir -p ~/.vman/tools
mkdir -p ~/.vman/shims
mkdir -p ~/.vman/cache

# Setup git hooks (if .git exists)
if [ -d ".git" ]; then
    echo "🪝 Setting up git hooks..."
    mkdir -p .git/hooks
    
    # Pre-commit hook
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# vman pre-commit hook

echo "Running pre-commit checks..."

# Run tests
if ! make test; then
    echo "❌ Tests failed"
    exit 1
fi

# Run linting
if ! make lint; then
    echo "❌ Linting failed"
    exit 1
fi

echo "✅ Pre-commit checks passed"
EOF
    
    chmod +x .git/hooks/pre-commit
    echo "✅ Git hooks configured"
fi

# Run initial tests
echo "🧪 Running initial tests..."
if make test; then
    echo "✅ All tests passed"
else
    echo "⚠️  Some tests failed, but environment setup is complete"
fi

echo ""
echo "🎉 Development environment setup complete!"
echo ""
echo "Available commands:"
echo "  make build    - Build the project"
echo "  make test     - Run tests"
echo "  make lint     - Run linting"
echo "  make clean    - Clean build artifacts"
echo "  make install  - Install vman locally"
echo ""
echo "Get started:"
echo "  ./build/vman --help"