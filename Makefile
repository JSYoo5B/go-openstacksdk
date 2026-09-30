.PHONY: test vet fmt check generate

generate:
	sh internal/cmd/sdkgen/generate.sh

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet test
	test -z "$$(gofmt -l .)"
