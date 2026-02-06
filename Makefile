BINARY := termdash
BUILD_DIR := bin

.PHONY: build run test clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/termdash

run: build
	./$(BUILD_DIR)/$(BINARY)

test:
	go test ./... -v

clean:
	rm -rf $(BUILD_DIR)
	go clean
