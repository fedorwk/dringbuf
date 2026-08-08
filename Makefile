.PHONY: all test bench run benchcmp fmt vet

all: test

test:
	go test ./... -count=1

bench:
	go test -run=^$$ -bench=. -benchmem -benchtime=300x

run: benchcmp

benchcmp:
	go run ./cmd/benchcmp

fmt:
	gofmt -l -w .

vet:
	go vet ./...
