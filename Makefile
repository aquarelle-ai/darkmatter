#!/usr/bin/make -f

PACKAGES=$(shell go list ./... | grep -v '/simulation')

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

export GO111MODULE = on

ldflags = -X cratos.network/darkmatter/version.Name=PriceTracker \
	-X cratos.network/darkmatter/version.Version=$(VERSION) \
	-X cratos.network/darkmatter/version.Commit=$(COMMIT) \
	-X "cratos.network/darkmatter/version.BuildTags=$(build_tags)"

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

all: lint install

include contrib/devtools/Makefile

build: go.sum
	@go build -mod=readonly ./...

install: go.sum
		go install -mod=readonly $(BUILD_FLAGS) ./app


########################################
### Tools & dependencies

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download
.PHONY: go-mod-cache

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify
	@go mod tidy

lint:
	golangci-lint run
	@find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -d -s
	go mod verify


########################################
### Documentation

godocs:
	@echo "--> Wait a few seconds and visit http://localhost:6060/pkg/github.com/cratos.network/darkmatter/types"
	godoc -http=:6060


test:
	@go test -mod=readonly $(PACKAGES)