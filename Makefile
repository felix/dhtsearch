
TARGETS = freebsd-amd64 linux-386 linux-amd64 linux-arm linux-arm64 darwin-amd64 windows-386 windows-amd64
CMD = dhtsearch
VERSION ?= $(shell git describe --tags --always)
SRC = $(shell find . -type f -name '*.go')
LDFLAGS = -ldflags="-w -s -X=main.version=$(VERSION)"
BINARIES = $(patsubst %,$(CMD)-%-v$(VERSION), $(TARGETS))

.DEFAULT_GOAL := help

release: $(BINARIES) ## Build all binaries

build: $(CMD) ## Build binary for current platform

$(CMD): check-env $(SRC)
	cd cmd && go build -o ../$(CMD) $(LDFLAGS)

standalone : TAGS = sqlite

$(BINARIES): check-env $(SRC)
	cd cmd && env GOOS=`echo $@ |cut -d'-' -f2` GOARCH=`echo $@ |cut -d'-' -f3 |cut -d'.' -f1` go build -o ../$@ $(LDFLAGS)

test: ## Run tests and create coverage report
	go test -short -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	@for file in $$(find . -name 'vendor' -prune -o -type f -name '*.go'); do golint $$file; done

clean: check-env ## Clean up temp files and binaries
	rm -f $(BINARIES)
	rm -f $(CMD)
	rm -rf coverage*

check-env:
ifndef VERSION
	$(error VERSION is undefined)
endif

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) |sort |awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: help install test lint clean
