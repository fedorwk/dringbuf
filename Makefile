.PHONY: all test fuzz bench run benchcmp fmt vet

all: test

test:
	go test -race ./... -count=1

fuzz:
	go test -run=^$$ -fuzz=FuzzBuffers -fuzztime=30s

bench:
	go test -run=^$$ -bench=. -benchmem -benchtime=300x

run: benchcmp

benchcmp:
	go run ./cmd/benchcmp

fmt:
	gofmt -l -w .

vet:
	go vet ./...
