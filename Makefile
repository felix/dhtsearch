
VERSION ?= $(shell git describe --tags --always)
SRC	:= $(shell find . -type f -name '*.go')
FLAGS	:= --tags fts5
PLAT	:= windows darwin linux freebsd openbsd
BINARY	:= $(patsubst %,dist/%,$(shell find cmd/* -maxdepth 0 -type d -exec basename {} \;))
RELEASE	:= $(foreach os, $(PLAT), $(patsubst %,%-$(os), $(BINARY)))

.PHONY: build
build: sqlite $(BINARY)

.PHONY: release
release: $(RELEASE)

dist/%: export GOOS=$(word 2,$(subst -, ,$*))
dist/%: bin=$(word 1,$(subst -, ,$*))
dist/%: $(SRC) $(shell find cmd/$(bin) -type f -name '*.go')
	go build -ldflags "-X main.version=$(VERSION)" $(FLAGS) \
	     -o $@ ./cmd/$(bin)

sqlite:
	CGO_ENABLED=1 go get -u $(FLAGS) github.com/mattn/go-sqlite3 \
		&& go install $(FLAGS) github.com/mattn/go-sqlite3

.PHONY: test
test:
	go test -short -coverprofile=coverage.out ./... \
		&& go tool cover -func=coverage.out

.PHONY: lint
lint: ; go vet ./...

.PHONY: clean
clean:
	rm -f coverage*
	rm -rf dist
