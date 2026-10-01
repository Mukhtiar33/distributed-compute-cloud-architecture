.PHONY: all build test lint clean setup

all: build

build:
	@echo "Building Control Plane..."
	cd control-plane && go build ./...
	@echo "Building Worker Agent..."
	cd worker-agent && go build ./...

test:
	@echo "Testing Control Plane..."
	cd control-plane && go test ./... -v
	@echo "Testing Worker Agent..."
	cd worker-agent && go test ./... -v

lint:
	@echo "Linting Control Plane..."
	cd control-plane && go vet ./...
	@echo "Linting Worker Agent..."
	cd worker-agent && go vet ./...

clean:
	rm -f control-plane/control-plane worker-agent/worker-agent

setup:
	./scripts/setup.sh
