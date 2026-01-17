.PHONY: help build install uninstall clean

BINARY_NAME=pace
INSTALL_PATH ?= $(HOME)/go/bin

help:
	@echo "Available commands:"
	@echo "  make build      Build the project"
	@echo "  make install    Install the project to $(INSTALL_PATH)"
	@echo "  make uninstall  Remove the installed binary"
	@echo "  make clean      Remove build artifacts"

build:
	@echo "Building the project..."
	go build -ldflags="-s -w" -o bin/$(BINARY_NAME) .
	@echo "Build completed. Binary is located at bin/$(BINARY_NAME)"

install: build
	@echo "Installing to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	cp bin/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation completed."

uninstall:
	@echo "Uninstalling..."
	rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstall completed."

clean:
	@echo "Cleaning..."
	rm -rf bin/
	@echo "Clean completed."
