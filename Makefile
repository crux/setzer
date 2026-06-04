# =============================================================
# Setzer — build orchestration
# =============================================================
BINARY  := setzer
VERSION := 0.1.0
APP     := build/Setzer.app
DIST    := dist

.DEFAULT_GOAL := help
.PHONY: help build run test fmt vet tidy clean app dist windows release

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
	@rm -rf "$(DIST)/dmgroot" && mkdir -p "$(DIST)/dmgroot"
	ditto "$(DIST)/Setzer.app" "$(DIST)/dmgroot/Setzer.app"
	ln -s /Applications "$(DIST)/dmgroot/Applications"
	hdiutil create -volname "Setzer" -srcfolder "$(DIST)/dmgroot" -ov -format UDZO "$(DIST)/Setzer-$(VERSION).dmg"
	@rm -rf "$(DIST)/dmgroot"
	@echo "==> $(DIST)/Setzer-$(VERSION).dmg"
	@shasum -a 256 "$(DIST)/Setzer-$(VERSION).dmg"

windows: ## Cross-build setzer.exe + NSIS installer (needs makensis; Linux/CI)
	@mkdir -p "$(DIST)"
	GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o "$(DIST)/setzer.exe" .
	makensis -DVERSION=$(VERSION) -DEXE="$(CURDIR)/$(DIST)/setzer.exe" -DOUT="$(CURDIR)/$(DIST)/Setzer-Setup-$(VERSION).exe" packaging/windows/setzer.nsi
	@echo "==> $(DIST)/Setzer-Setup-$(VERSION).exe"

release: ## Tag & push a release; CI builds it (usage: make release VERSION=x.y.z)
	@test "$(origin VERSION)" = "command line" || { echo "usage: make release VERSION=x.y.z"; exit 1; }
	@test "$$(git rev-parse --abbrev-ref HEAD)" = "main" || { echo "error: releases are cut from main"; exit 1; }
	@git diff-index --quiet HEAD -- || { echo "error: working tree not clean — commit first"; exit 1; }
	@test -z "$$(git log origin/main..HEAD --oneline 2>/dev/null)" || { echo "error: unpushed commits on main — push first"; exit 1; }
	@if git rev-parse --verify "refs/tags/v$(VERSION)" >/dev/null 2>&1; then echo "error: tag v$(VERSION) already exists"; exit 1; fi
	git tag -a "v$(VERSION)" -m "v$(VERSION)"
	git push origin "v$(VERSION)"
	@echo "==> tagged v$(VERSION) — CI is building the release (gh run watch)"

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
