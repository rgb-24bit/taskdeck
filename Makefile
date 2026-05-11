.PHONY: build install clean run

BINARY=td
BUILD_DIR=bin
INSTALL_PATH=$(HOME)/.local/bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/td

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_PATH)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY) serve
