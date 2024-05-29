# gourl - Extract URLs from STDIN
# See LICENSE file for copyright and license details.

NAME=gourl
SRC=./main.go
GOBIN=./bin
BIN=$(GOBIN)/$(NAME)
PREFIX?=/usr/local
INSTALL_DIR=$(PREFIX)/bin

.PHONY: all build run vet clean test full

all: full

full: vet build

build: vet ## Generate bin
	@echo '>> Building $(NAME)'
	go build -ldflags "-s -w" -o $(BIN) $(SRC)

debug: vet ## Generate bin with debugger
	@echo '>> Building $(NAME) with debugger'
	@go build -gcflags="all=-N -l" -o $(BIN)-debug $(SRC)

run: build ## Run
	@echo '>> Running $(NAME)'
	$(BIN)

vet: ## Lint
	@echo '>> Checking code with go vet'
	@go vet ./...

clean: ## Clean
	@echo '>> Cleaning up'
	rm -rf $(GOBIN)
	go clean -cache

install: ## Install on system
	mkdir -p $(INSTALL_DIR)
	cp $(BIN) $(INSTALL_DIR)/$(NAME)
	chmod 755 $(INSTALL_DIR)/$(NAME)
	@echo '>> $(NAME) has been installed on your device'

uninstall: ## Uninstall from system
	rm -rf $(GOBIN)
	rm -rf $(INSTALL_DIR)/$(NAME)
	@echo '>> $(NAME) has been removed from your device'
