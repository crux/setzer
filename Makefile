# =============================================================
# Setzer — build orchestration
# =============================================================
BINARY  := setzer
VERSION := 0.1.0
APP     := build/Setzer.app
DIST    := dist

.DEFAULT_GOAL := help
.PHONY: help build run test fmt vet tidy clean app dist

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

dist: ## Build a universal Setzer.app DMG for release (-> dist/)
	@rm -rf "$(DIST)/Setzer.app" "$(DIST)/setzer-arm64" "$(DIST)/setzer-amd64" "$(DIST)/Setzer-$(VERSION).dmg"
	@mkdir -p "$(DIST)/Setzer.app/Contents/MacOS" "$(DIST)/Setzer.app/Contents/Resources"
	GOOS=darwin GOARCH=arm64 go build -o "$(DIST)/setzer-arm64" .
	GOOS=darwin GOARCH=amd64 go build -o "$(DIST)/setzer-amd64" .
	lipo -create -output "$(DIST)/Setzer.app/Contents/MacOS/setzer" "$(DIST)/setzer-arm64" "$(DIST)/setzer-amd64"
	cp packaging/macos/Info.plist "$(DIST)/Setzer.app/Contents/Info.plist"
	/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(VERSION)" "$(DIST)/Setzer.app/Contents/Info.plist"
	/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $(VERSION)" "$(DIST)/Setzer.app/Contents/Info.plist"
	@rm -f "$(DIST)/setzer-arm64" "$(DIST)/setzer-amd64"
	hdiutil create -volname "Setzer" -srcfolder "$(DIST)/Setzer.app" -ov -format UDZO "$(DIST)/Setzer-$(VERSION).dmg"
	@echo "==> $(DIST)/Setzer-$(VERSION).dmg"
	@shasum -a 256 "$(DIST)/Setzer-$(VERSION).dmg"

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
	rm -rf $(BINARY) build $(DIST)
