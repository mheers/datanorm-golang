.PHONY: test race vet fmt build clean

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

build:
	go build -o bin/datanorm ./cmd/datanorm

clean:
	rm -rf bin
