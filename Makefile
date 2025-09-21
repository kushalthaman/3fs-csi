SHELL := /bin/bash
GOOS ?= linux
GOARCH ?= amd64
BIN_DIR := bin
BIN := $(BIN_DIR)/3fs-csi-node
PKG := github.com/3fs-csi/3fs-csi

.PHONY: all build clean test image

all: build

$(BIN):
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/node

build: $(BIN)

clean:
	rm -rf $(BIN_DIR)

test:
	go test ./...

image:
	docker build -t 3fs-csi/node:dev -f images/Dockerfile .


