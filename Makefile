# GuardCompress Makefile (Windows PowerShell + *nix compatible via Go)
# Usage: make build | make test | make release

BINARY_NAME=guardcompress
CORE_DIR=core
BIN_DIR=core/bin

.PHONY: build test clean release

build:
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./$(CORE_DIR)
	@echo "Built: $(BIN_DIR)/$(BINARY_NAME)"

# Windows explicit
build-win:
	go build -o $(CORE_DIR)/bin/$(BINARY_NAME)-windows-amd64.exe ./$(CORE_DIR)

test:
	go test ./$(CORE_DIR)/... -v

clean:
	rm -rf $(BIN_DIR) tmp/out

release:
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CORE_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CORE_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CORE_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CORE_DIR)
	@echo "Release binaries in $(BIN_DIR)/"
