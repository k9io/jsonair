BINARY_DIR=bin
MODULE_NAME=github.com/k9io/jsonair

.PHONY: all build build-server build-client clean test tidy

# Default action when you just type 'make'

#all: tidy build test
all: tidy build

# Create the bin directory and build everything

build: jsonair jsonair-agent

jsonair:
	
	@echo "Building JSONAir....."
	go build -o $(BINARY_DIR)/jsonair/jsonair ./cmd/jsonair

jsonair-agent:

	@echo "Building JSONAir Agent....."
	go build -o $(BINARY_DIR)/jsonair-agent/jsonair-agent ./cmd/jsonair-agent


build-all:

#	@echo "Building for Windows..."

#	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/highvolt-server/windows/highvolt-server.exe ./cmd/highvolt-server

#	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/clients/suricata/windows/suricata.exe ./cmd/clients/suricata
#	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/clients/aws-s3/windows/aws-s3.exe ./cmd/aws-s3


#	@echo "Building for Linux..."
#	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/linux/server ./cmd/server
#	#GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/linux/client ./cmd/client

#	@echo "Building for macOS (Intel & Apple Silicon)..."
#	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BINARY_DIR)/mac/server-intel ./cmd/server
	#GOOS=darwin GOARCH=arm64 go build -o $(BINARY_DIR)/mac/server-m1 ./cmd/server


# Clean up build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)

# Run all tests (including shared code)
#test:
#	go test ./...

# Tidy up the go.mod file
tidy:
	go mod tidy

