BINARY := bin/server
MODULE  := github.com/Tarasa24/psp-integration-demo
LDFLAGS := -s -w

.PHONY: build test lint fmt docker-up clean

build:
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/server

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

clean:
	rm -rf bin/
	go clean -testcache
