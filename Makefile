.PHONY: all build clean run install deps tidy help

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@echo "  build   - Build the lltop binary"
	@echo "  clean   - Remove built binaries"
	@echo "  run     - Build and launch lltop"
	@echo "  install - Install lltop to ~/.local/bin"
	@echo "  deps    - Download Go dependencies"
	@echo "  tidy    - Run go mod tidy"
	@echo "  help    - Show this help message"

all: build

build:
	go build -o bin/lltop ./cmd/lltop

clean:
	rm -rf bin/

run: build
	./bin/lltop

install: build
	cp bin/lltop ~/.local/bin/

deps:
	go mod download

tidy:
	go mod tidy
