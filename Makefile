.PHONY: build install lint test clean

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
