NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
APP=ferry
REVISION=$(shell git rev-parse --short HEAD)
BASE_VERSION=$(shell cat VERSION)
VERSION=$(BASE_VERSION)-$(REVISION)

all: build

install-deps:
	@script/install-deps

image:
	@script/image $(APP) $(VERSION)

build:
	@echo "$(OK_COLOR)==> Building revision $(VERSION)...$(NO_COLOR)"
	@script/build $(APP) $(VERSION)

release: ci
	@echo "$(OK_COLOR)==> Building revision $(VERSION)...$(NO_COLOR)"
	@./script/wrap-release-docker $(APP) $(VERSION)

format:
	goop go fmt ./...

test:
	@echo "$(OK_COLOR)==> Testing...$(NO_COLOR)"
	@script/test

bench:
	LOG_LEVEL=QUIET goop go test -run=NOTESTS -bench=. -benchmem ./...

ci: image
	@echo "$(OK_COLOR)==> Testing...$(NO_COLOR)"
	@script/ci $(APP) $(VERSION)

run: build
	@echo "$(OK_COLOR)==> Running...$(NO_COLOR)"
	@sudo build/$(BIN_NAME)

.PHONY: all install-deps build release format test bench ci run image
