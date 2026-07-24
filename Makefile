.PHONY: check format-check module-check vet test integration e2e build build-target build-all workflow-check

check: format-check module-check vet test integration e2e build

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

integration:
	go test -race -count=1 ./internal/apply

e2e: build
	bash ./scripts/ci/cli-smoke.sh ./bin/repoctl

build:
	mkdir -p bin
	go build -trimpath -o ./bin/repoctl ./cmd/repoctl

build-target:
	@test -n "$(GOOS)" || { echo 'GOOS is required' >&2; exit 2; }
	@test -n "$(GOARCH)" || { echo 'GOARCH is required' >&2; exit 2; }
	mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -trimpath -o ./bin/repoctl-$(GOOS)-$(GOARCH) ./cmd/repoctl

build-all:
	$(MAKE) build-target GOOS=linux GOARCH=amd64
	$(MAKE) build-target GOOS=windows GOARCH=amd64
	$(MAKE) build-target GOOS=darwin GOARCH=amd64

workflow-check:
	go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
	python3 ./scripts/ci/workflow-policy.py
