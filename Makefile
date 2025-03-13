.PHONY: build test clean run

# Default target
all: build

# Build the application
build:
	@./build.sh

# Run tests
test:
	@./test.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f concap
	@echo "Clean completed."

# Run the application with example directory
run:
	@echo "Running concap with example directory..."
	@./concap --dir ./example

# Help target
help:
	@echo "Available targets:"
	@echo "  build  - Build the application"
	@echo "  test   - Run tests"
	@echo "  clean  - Clean build artifacts"
	@echo "  run    - Run the application with example directory"
	@echo "  help   - Show this help message" 