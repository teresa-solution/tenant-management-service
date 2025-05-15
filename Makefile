# Makefile for Go application

# Default target when just typing 'make'
.PHONY: all
all: server

# Run the server
.PHONY: server
server:
	go run cmd/server/main.go

# Run database migrations up
.PHONY: migrate-up
migrate-up:
	go run cmd/migrate/main.go -command up

# Run database migrations down
.PHONY: migrate-down
migrate-down:
	go run cmd/migrate/main.go -command down

# Run both server and migrations in sequence
.PHONY: run-all
run-all: migrate-up server

# Clean any build artifacts (add specific clean steps as needed)
.PHONY: clean
clean:
	rm -f *.out

# Help target to display available commands
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make server      - Run the server"
	@echo "  make migrate-up  - Run database migrations up"
	@echo "  make migrate-down- Run database migrations down"
	@echo "  make run-all     - Run migrations up and then start the server"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make help        - Show this help message"
