.PHONY: test vet fmt check

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet test
	test -z "$$(gofmt -l .)"
