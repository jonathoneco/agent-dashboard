.PHONY: help build install lint test clean

help:
	@printf "Targets:\n"
	@printf "  build    Build ./agent-dashboard\n"
	@printf "  install  Build and refresh ~/.local/bin/agent-dashboard symlink\n"
	@printf "  test     Run go test ./...\n"
	@printf "  lint     Run go vet and gofmt -l\n"
	@printf "  clean    Remove ./agent-dashboard\n"

build:
	go build -o agent-dashboard ./cmd/dashboard

install: build
	ln -sf $(CURDIR)/agent-dashboard ~/.local/bin/agent-dashboard

test:
	go test ./...

lint:
	go vet ./...
	gofmt -l .

clean:
	rm -f agent-dashboard
