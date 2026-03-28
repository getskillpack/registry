.PHONY: test e2e-smoke build

build:
	go build -o registry ./cmd/registry

test:
	go test ./... -count=1

e2e-smoke:
	bash scripts/e2e-smoke.sh
