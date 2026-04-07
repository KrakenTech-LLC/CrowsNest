# Makefile for CrowsNest

# Go command
GO=go

# Binary name
BINARY_NAME=crowsnest

# Build directory
BUILD_DIR=build/bin

# Platforms to build for
PLATFORMS=linux darwin windows

# Architecture to build for
ARCHS=amd64 arm64

# Version info from git tag or default
VERSION=$(shell git describe --tags 2>/dev/null || echo "v1.3.3")

.PHONY: all clean build build-all

# Default target
all: clean build-all

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)

# Build for current platform
build:
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags "-X main.version=$(VERSION) -s -w" crowsnest.go

# Build for all platforms
build-all: clean
	@for platform in $(PLATFORMS); do \
	    for arch in $(ARCHS); do \
	        echo "Building for $$platform/$$arch..."; \
	        GOOS=$$platform GOARCH=$$arch CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-$$platform-$$arch -ldflags "-X main.version=$(VERSION) -s -w" crowsnest.go; \
	        if [ "$$platform" = "windows" ]; then \
	            mv $(BUILD_DIR)/$(BINARY_NAME)-$$platform-$$arch $(BUILD_DIR)/$(BINARY_NAME)-$$platform-$$arch.exe; \
	        fi; \
	    done; \
	done

# Install locally
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Run tests
test:
	$(GO) test ./...
