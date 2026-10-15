# lazyflk: `make` lists what you can do.

BIN := lazyflk
LINK := $(HOME)/.local/bin/lazyflk

.PHONY: help build install update run demo test check snapshots clean

help: ## what you can do (this list)
	@echo "lazyflk: make TARGET"
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | sed 's/:.*## /\t/' | awk -F'\t' '{printf "  make %-10s %s\n", $$1, $$2}'
	@echo "Installed: $$(readlink -f $(LINK) 2>/dev/null || echo 'not yet (make install)')"

build: ## build ./lazyflk from this checkout
	go build -o $(BIN) ./cmd/lazyflk
	@./$(BIN) --version

install: build ## build, and link it as ~/.local/bin/lazyflk (once; then build is enough)
	@mkdir -p $(dir $(LINK))
	ln -sfn $(CURDIR)/$(BIN) $(LINK)
	@echo "lazyflk -> $(CURDIR)/$(BIN)"

update: ## git pull, then build (what's new: git log)
	git pull --ff-only
	$(MAKE) build

run: build ## build and run against this host's flk
	./$(BIN)

demo: build ## build and run on the captured fixtures: no flk calls, actions run nothing
	./$(BIN) --demo

test: ## run the tests
	go test ./...

check: ## what CI runs: gofmt, vet, build, race tests
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt: run gofmt -w on these"; exit 1; }
	go vet ./...
	go build -o $(BIN) ./cmd/lazyflk
	@if command -v gcc >/dev/null; then CGO_ENABLED=1 go test -race ./...; \
	else echo "no C compiler here: tests without -race (CI runs them with it)"; go test ./...; fi

snapshots: ## rewrite the screen snapshots after an intended change (review with git diff)
	go test ./internal/ui -update
	@git diff --stat -- internal/ui/testdata

clean: ## remove the built binary
	rm -f $(BIN)
