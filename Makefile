BINARY_NAME = digler
MAIN_FILE = cmd/main.go
OUTPUT_DIR = bin

MODULE := $(shell go list -m)
ENV_PKG = $(MODULE)/internal/env

# Target platforms: os/arch
TARGETS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Get the latest tag (if any)
TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo "")

# Get the current branch name
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

# Get the short commit hash
SHORT_COMMIT := $(shell git rev-parse --short HEAD)

# Compute version: if tag exists use tag, otherwise branch-name-short-commit
ifeq ($(TAG),)
	VERSION := $(BRANCH)-$(SHORT_COMMIT)
else
	VERSION := $(TAG)
endif

# Get the full commit hash
COMMIT_HASH := $(shell git rev-parse HEAD)

# Get build time in ISO8601 format
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

.PHONY: all build clean version

all: build

build:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Building $(BINARY_NAME) version: $(VERSION)"
	@for target in $(TARGETS); do \
		GOOS=$${target%%/*} && GOARCH=$${target##*/}; \
		output_name="$(BINARY_NAME)-$${GOOS}-$${GOARCH}"; \
		if [ "$${GOOS}" = "windows" ]; then output_name="$$output_name.exe"; fi; \
		echo "-> $$output_name"; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-X $(ENV_PKG).Version=$(VERSION) -X $(ENV_PKG).CommitHash=$(COMMIT_HASH) -X $(ENV_PKG).BuildTime=$(BUILD_TIME)" -o $(OUTPUT_DIR)/$$output_name $(MAIN_FILE); \
	done

clean:
	rm -rf $(OUTPUT_DIR)

version:
	@echo $(VERSION)
