
BINARY=dhtsearch
VERSION=$(shell git describe --tags --always)
SRC=$(shell find . -type f -name '*.go')

build: $(BINARY)

$(BINARY): $(SRC)
	cd cmd && go build -ldflags "-w -s \
		-X main.version=$(VERSION)" \
		-o ../$(BINARY)
test:
	go test -short -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	@for file in $$(find . -name 'vendor' -prune -o -type f -name '*.go'); do golint $$file; done

clean:
	rm -f $(BINARY)
	rm -rf coverage*

.PHONY: install build test lint clean
