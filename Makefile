NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

BASE_VERSION=$(shell cat VERSION)
REVISION=$(shell git rev-parse --short HEAD)
VERSION=$(BASE_VERSION)-$(REVISION)

all: install

install-deps:
	@echo "$(OK_COLOR)==> Installing dependencies...$(NO_COLOR)"
	go get github.com/dqminh/govendor
	govendor deps.json

build:
	@echo "$(OK_COLOR)==> Building version $(VERSION)...$(NO_COLOR)"
	go build -ldflags "-X main.version $(VERSION)" -v ./...

install:
	@echo "$(OK_COLOR)==> Installing version $(VERSION)...$(NO_COLOR)"
	go install -ldflags "-X main.version $(VERSION)" -v ./...

clean:
	go clean -i -x ./...

test:
	ginkgo -r --trace
