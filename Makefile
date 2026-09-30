.PHONY: test vet fmt check generate parity

generate:
	sh internal/cmd/sdkgen/generate.sh

test:
	go test -race -timeout 60s ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

parity:
	go run ./internal/cmd/paritycheck

check: vet test parity
	test -z "$$(gofmt -l .)"
