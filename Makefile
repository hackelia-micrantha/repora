.PHONY: check format-check module-check vet test integration e2e build build-all

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
	./scripts/ci/cli-smoke.sh ./bin/repoctl

build:
	mkdir -p bin
	go build -trimpath -o ./bin/repoctl ./cmd/repoctl

build-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath ./...
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath ./...
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath ./...
