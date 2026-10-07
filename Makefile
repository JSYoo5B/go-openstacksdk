.PHONY: test vet fmt check generate parity progress progress-check smoke

generate:
	sh internal/cmd/sdkgen/generate.sh

test:
	go test -race -timeout 60s ./...

smoke:
	python3 internal/cmd/parity/smoke.py

vet:
	go vet ./...

fmt:
	gofmt -w .

parity:
	go run ./internal/cmd/paritycheck

progress:
	python3 internal/cmd/parity/progress.py

progress-check:
	python3 internal/cmd/parity/progress.py --check

check: vet test parity progress-check
	test -z "$$(gofmt -l .)"
