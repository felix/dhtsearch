
VERSION	!= git describe --tags --always
SRC	!= find . -type f -name '*.go'
PLAT	:= windows darwin linux freebsd openbsd
BINARY	:= $(patsubst %,dist/%,$(shell find cmd/* -maxdepth 0 -type d -exec basename {} \;))
RELEASE	:= $(foreach os, $(PLAT), $(patsubst %,%-$(os), $(BINARY)))

.PHONY: build
build: $(BINARY)

.PHONY: release
release: $(RELEASE)

dist/%: export GOOS=$(word 2,$(subst -, ,$*))
dist/%: bin=$(word 1,$(subst -, ,$*))
dist/%: $(SRC) $(shell find cmd/$(bin) -type f -name '*.go')
	go build -ldflags "-X main.version=$(VERSION)" \
	     -o $@ ./cmd/$(bin)

.PHONY: test
test:
	go test -short -coverprofile=coverage.out ./... \
		&& go tool cover -func=coverage.out

.PHONY: lint
lint: ; golangci-lint run

.PHONY: clean
clean:
	rm -f coverage*
	rm -rf dist
