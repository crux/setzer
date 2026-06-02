# =============================================================
# Setzer — build orchestration
# =============================================================
BINARY := setzer

APP := build/Setzer.app

.DEFAULT_GOAL := help
.PHONY: help build run test fmt vet tidy clean app

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[1m%-8s\033[0m %s\n", $$1, $$2}'

build: ## Compile the setzer binary
	go build -o $(BINARY) .

app: ## Build Setzer.app (macOS, host arch; double-click to run)
	@rm -rf "$(APP)"
	@mkdir -p "$(APP)/Contents/MacOS" "$(APP)/Contents/Resources"
	go build -o "$(APP)/Contents/MacOS/setzer" .
	cp packaging/macos/Info.plist "$(APP)/Contents/Info.plist"
	@echo "built $(APP) — run: open $(APP)"

run: build ## Build and run (serves http://127.0.0.1:8765)
	./$(BINARY)

test: ## Run the test suite
	go test ./...

fmt: ## Format the source
	go fmt ./...

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod / go.sum
	go mod tidy

clean: ## Remove build artifacts
	rm -f $(BINARY)
