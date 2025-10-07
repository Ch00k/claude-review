.PHONY: lint test test-verbose test-one test-ci air build build-release dev release release-patch release-minor release-major install-slash-commands install-hooks install

.EXPORT_ALL_VARIABLES:

CGO_ENABLED = 1
CR_EXECUTABLE_FILENAME ?= claude-review
CR_BUILD_ARTIFACTS_DIR ?= dist
CR_VERSION ?= dev


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
	go run gotest.tools/gotestsum@latest --format testname -- -coverprofile=coverage.txt -tags e2e ./...

air:
	air -c .air.toml

build:
	go build -trimpath -ldflags="-s -w -X main.Version=${CR_VERSION}" -o ./${CR_BUILD_ARTIFACTS_DIR}/${CR_EXECUTABLE_FILENAME} .

build-release: build

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

install: build
	./${CR_BUILD_ARTIFACTS_DIR}/${CR_EXECUTABLE_FILENAME} install
