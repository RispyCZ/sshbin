BINARY_NAME=sshbin
MAIN_PATH=./cmd/sshbin
GO=go

dev:
	watchexec -r -e go,js,css,html go run ${MAIN_PATH} --storage /tmp/${BINARY_NAME}

build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run:
	$(GO) run $(MAIN_PATH)

test:
	$(GO) test ./...

test-coverage:
	$(GO) test -cover ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

deps:
	$(GO) mod download
	$(GO) mod tidy

clean:
	$(GO) clean
	rm -rf bin/

.PHONY: build run test test-coverage fmt vet deps clean