BINARY_NAME = digler
MAIN_FILE = cmd/main.go
OUTPUT_DIR = bin

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
	go build -ldflags "-X main.Version=$(VERSION) -X main.CommitHash=$(COMMIT_HASH) -X main.BuildTime=$(BUILD_TIME)" -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_FILE)

clean:
	rm -rf $(OUTPUT_DIR)

version:
	@echo $(VERSION)
