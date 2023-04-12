.PHONY: build test setup

BINARY_NAME = shaka-streamer
BUILD_DIR = $(PWD)/build
BINARY_PATH = $(BUILD_DIR)/$(BINARY_NAME)

build:
	go build -o $(BINARY_PATH) shaka-streamer.go

test: 
	go test -v -cover ./...

setup: 
	go run shaka-streamer.go --setup