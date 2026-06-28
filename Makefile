BINARY_NAME=sshbin
MAIN_PATH=./cmd/sshbin
GO=go
DEV_BIN=/tmp/sshbin-dev
export AWS_ENDPOINT_URL=http://localhost:9090
export AWS_ACCESS_KEY_ID=dev
export AWS_SECRET_ACCESS_KEY=dev
export AWS_REGION=us-east-1

dev:
	watchexec -r -e go,js,css,html 'go build -o $(DEV_BIN) $(MAIN_PATH) && exec $(DEV_BIN) --dev --storage s3://sshbin'

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

.PHONY: build run dev test test-coverage fmt vet deps clean