# go-extract-url - Extract URLs from STDIN and show them in dmenu


NAME = gourl
SRC = ./main.go
BIN = ./bin/$(NAME)

.PHONY: all build run vet clean

all: full

full: vet lint build

help:	## This help dialog.
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

build: vet ## Generate bin
	@echo '>> Building $(NAME)'
	@go build -ldflags "-s -w" -o $(BIN) $(SRC)

debug: vet test ## Generate bin with debugger
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
	rm -f $(BIN)
	go clean -cache

.PHONY: lint
lint: vet
	@echo '>> Linting code'
	@golangci-lint run ./...
	@codespell ./main.go

.PHONY: check
check: ## Lint all
	@echo '>> Linting everything'
	@golangci-lint run -p bugs -p error
