
TARGETS = freebsd-amd64 linux-386 linux-amd64 linux-arm linux-arm64 darwin-amd64 windows-386 windows-amd64
CMD = dhtsearch
VERSION ?= $(shell git describe --tags --always)
SRC = $(shell find . -type f -name '*.go')
FLAGS = --tags fts5
BINARIES = $(patsubst %,$(CMD)-%-v$(VERSION), $(TARGETS))

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		|sort \
		|awk 'BEGIN{FS=":.*?## "};{printf "\033[36m%-30s\033[0m %s\n",$$1,$$2}'

.PHONY: build
build: sqlite $(BINARIES) ## Build all binaries

$(BINARIES): $(SRC)
	cd cmd && env GOOS=`echo $@ |cut -d'-' -f2` \
		GOARCH=`echo $@ |cut -d'-' -f3 |cut -d'.' -f1` \
		go build -o ../$@ $(FLAGS) -ldflags="-w -s -X=main.version=$(VERSION)"

sqlite:
	go get -u $(FLAGS) github.com/mattn/go-sqlite3 \
		&& go install $(FLAGS) github.com/mattn/go-sqlite3

.PHONY: test
test: ## Run tests and create coverage report
	go test -short -coverprofile=coverage.out ./... \
		&& go tool cover -func=coverage.out

.PHONY: lint
lint:
	revive ./...

.PHONY: clean
clean: ## Clean up temp files and binaries
	rm -f $(CMD)-*-v*
	rm -rf coverage*
