.PHONY: lint test test-verbose test-one test-ci air build assets build-release dev release release-patch release-minor release-major install-slash-commands

.EXPORT_ALL_VARIABLES:

CGO_ENABLED = 1
CR_EXECUTABLE_FILENAME ?= claude-review
CR_WEB_ASSETS_FILENAME ?= frontend-assets.tar.gz
CR_BUILD_ARTIFACTS_DIR ?= dist


prettier:
	prettier --write frontend/

lint: prettier
	golangci-lint run --fix

test:
	gotestsum --format testname ./...

test-verbose:
	gotestsum --format standard-verbose -- -v -count=1 ./...

test-one:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-one TEST=TestName"; \
		exit 1; \
	fi
	gotestsum --format standard-verbose -- -v -count=1 -run "^$(TEST)$$" ./...

test-ci:
	go run gotest.tools/gotestsum@latest --format testname -- -coverprofile=coverage.txt ./...

air:
	air -c .air.toml

build:
	go build -o ./${CR_BUILD_ARTIFACTS_DIR}/${CR_EXECUTABLE_FILENAME} .

assets:
	tar -czf ./${CR_BUILD_ARTIFACTS_DIR}/${CR_WEB_ASSETS_FILENAME} frontend/

build-release: build assets

dev: air

release:
	@echo "Available release types:"
	@echo "  make release-patch  # Patch version (x.y.Z)"
	@echo "  make release-minor  # Minor version (x.Y.0)"
	@echo "  make release-major  # Major version (X.0.0)"

release-patch:
	./release.sh patch

release-minor:
	./release.sh minor

release-major:
	./release.sh major

install-slash-commands:
	mkdir -p ~/.claude/commands
	cp slash-commands/address-comments.md ~/.claude/commands/address-comments.md
