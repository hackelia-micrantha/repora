VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || printf unknown)
BUILD_LDFLAGS ?= -X main.version=$(VERSION) -X main.commit=$(COMMIT)
GITLEAKS_VERSION ?= v8.30.1
GO_LICENSES_VERSION ?= v1.6.0

.PHONY: check format-check module-check vet test coverage integration route-test receipt-test e2e build build-target build-all workflow-check deep-repeat deep-integration security-secrets security-licenses release-package release-verify

check: format-check module-check vet test integration route-test receipt-test e2e build

format-check:
	@files="$$(find . -name '*.go' -not -path './.git/*' -exec gofmt -l {} +)"; \
	if [ -n "$$files" ]; then printf '%s\n' "$$files"; exit 1; fi

module-check:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

vet:
	go vet ./...

test:
	go test -race -count=1 -short ./...

coverage:
	mkdir -p artifacts/coverage
	go test -race -count=1 -short -covermode=atomic \
		-coverprofile=artifacts/coverage/coverage.out ./...
	go tool cover -func=artifacts/coverage/coverage.out | \
		tee artifacts/coverage/coverage.txt

integration:
	go test -race -count=1 ./internal/apply

route-test:
	python3 ./scripts/ci/validate_manifest_paths.py \
		./.repora/document-router.yaml
	go run ./scripts/ci/route-tests.go \
		./.repora/document-router.yaml ./.repora/route-tests.json
	python3 ./scripts/ci/trust-policy.py \
		./.repora/document-router.yaml ./.repora/trust-tests.json

receipt-test:
	python3 ./scripts/ci/context-receipt.py \
		./examples/context-receipt-v1.json
	python3 -m unittest discover -s scripts/ci -p 'test_context_receipt.py'

deep-repeat:
	REPEAT_COUNT="$${REPEAT_COUNT:-10}" bash ./scripts/ci/repeat-tests.sh ./...

deep-integration:
	go test -race -count=1 ./...

e2e: build
	bash ./scripts/ci/cli-smoke.sh ./bin/repoctl

build:
	mkdir -p bin
	go build -trimpath -buildvcs=false -ldflags "$(BUILD_LDFLAGS)" -o ./bin/repoctl ./cmd/repoctl

build-target:
	@test -n "$(GOOS)" || { echo 'GOOS is required' >&2; exit 2; }
	@test -n "$(GOARCH)" || { echo 'GOARCH is required' >&2; exit 2; }
	mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -trimpath -buildvcs=false -ldflags "$(BUILD_LDFLAGS)" \
		-o ./bin/repoctl-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,) \
		./cmd/repoctl

build-all:
	$(MAKE) build-target GOOS=linux GOARCH=amd64
	$(MAKE) build-target GOOS=windows GOARCH=amd64
	$(MAKE) build-target GOOS=darwin GOARCH=amd64
	$(MAKE) build-target GOOS=darwin GOARCH=arm64

security-secrets:
	go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION) \
		git --redact --no-banner --no-color .

security-licenses:
	mkdir -p artifacts/security
	go run github.com/google/go-licenses@$(GO_LICENSES_VERSION) \
		check ./cmd/repoctl --ignore repoctl
	go run github.com/google/go-licenses@$(GO_LICENSES_VERSION) \
		report ./cmd/repoctl --ignore repoctl | \
		LC_ALL=C sort > artifacts/security/licenses.csv
	test -s artifacts/security/licenses.csv

release-package:
	bash ./scripts/release/package.sh

release-verify:
	bash ./scripts/release/verify.sh

workflow-check:
	go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
	python3 -m unittest discover -s scripts/ci -p 'test_*.py'
	python3 ./scripts/ci/workflow-policy.py
