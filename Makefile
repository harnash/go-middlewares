SOURCEDIR := .
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
# Go utilities
ifeq ($(OS),Windows_NT)
	GO_PATH := $(subst \,/,${GOPATH})
else
	GO_PATH := ${GOPATH}
endif

ifeq ($(GO_PATH),)
    GO_PATH := $(shell go env GOPATH)
endif

# Remove the trailing slash
GO_PATH := $(patsubst %/,%,$(GO_PATH))

# GO_PATH := $(realpath $(GO_PATH))

ifeq ($(OS),Windows_NT)
	BINARY_EXT := .exe
else
	BINARY_EXT :=
endif

GO_LINT := $(GO_PATH)/bin/gometalinter$(BINARY_EXT)
GO_BINDATA := $(GO_PATH)/bin/go-bindata$(BINARY_EXT)
GO_COV := $(GO_PATH)/bin/gocov$(BINARY_EXT)
GO_COV_XML := $(GO_PATH)/bin/gocov-xml$(BINARY_EXT)
GO_GOVER := $(GO_PATH)/bin/gover$(BINARY_EXT)
GO_GINKGO := $(GO_PATH)/bin/ginkgo$(BINARY_EXT)

# Handling project dirs and names
ROOT_DIR := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ifeq ($(OS),Windows_NT)
	ROOT_DIR := $(subst \,/,${ROOT_DIR})
endif
PROJECT_PATH := $(shell go list .)
PROJECT_NAME := $(lastword $(subst /, , $(PROJECT_PATH)))

BINARY := bin/$(PROJECT_NAME)$(BINARY_EXT)

TARGETS := $(shell go list ./... | grep -v ^$(PROJECT_PATH)/vendor | sed s!$(PROJECT_PATH)/!! | grep -v $(PROJECT_PATH))
FQN_TARGETS := $(shell go list ./... | grep -v ^$(PROJECT_PATH)/vendor | grep -ve ^$(PROJECT_PATH)$)
TARGETS_LINT := $(patsubst %,lint-%, $(TARGETS))
TARGETS_VET  := $(patsubst %,vet-%, $(TARGETS))
TARGETS_FMT  := $(patsubst %,fmt-%, $(TARGETS))

# Injecting project version and build time
ifeq ($(OS),Windows_NT)
	VERSION_GIT := $(shell cmd /C 'git describe --always --tags --abbrev=7')
else
	VERSION_GIT := $(shell sh -c 'git describe --always --tags --abbrev=7')
endif
ifeq ($(VERSION_GIT),)
	VERSION_GIT = "v0.0.0"
endif

ifeq ($(OS),Windows_NT)
	BUILD_TIME := $(shell PowerShell -Command "get-date -format yyyy-MM-ddTHH:mm:SSzzz")
else
	BUILD_TIME := `date +%FT%T%z`
endif

VERSION_PACKAGE := $(PROJECT_PATH)/common
LDFLAGS := -ldflags "-X $(VERSION_PACKAGE).Version=${VERSION_GIT} -X $(VERSION_PACKAGE).BuildTime=${BUILD_TIME}"

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build ${LDFLAGS} -o ${BINARY} main.go

$(GO_LINT):
	curl -L https://git.io/vp6lP | sh -s -- -b ${GO_PATH}/bin -d latest

$(GO_GOVER):
	go get github.com/modocache/gover

$(GO_COV):
	go get github.com/axw/gocov/gocov

$(GO_COV_XML):
	go get github.com/t-yuki/gocov-xml

$(GO_GINKGO):
	go get github.com/onsi/ginkgo/ginkgo

prepare: $(GO_DEP)
	go mod download

install: $(BINARY)
	go install ${LDFLAGS} ./...

run: $(BINARY)
	@if test -f .ENV ; then source .ENV; fi
	@$(BINARY)

test: vet $(GO_GINKGO)
	$(GO_GINKGO) -r -randomizeAllSpecs -randomizeSuites -failOnPending -trace -race -keepGoing -flakeAttempts 2

test-cover: $(GO_COV) $(GO_COV_XML) $(GO_GINKGO) $(GO_GOVER)
	$(GO_GINKGO) -r -randomizeAllSpecs -randomizeSuites -failOnPending -cover -trace --race -keepGoing
	@gover
	@gocov convert gover.coverprofile | gocov-xml > coverage.xml

vet: $(TARGETS_VET)
# @go vet

$(TARGETS_VET): vet-%: %
	@go vet $(PROJECT_PATH)/$</

fmt-check:
	@test -z "$$(gofmt -s -l $(TARGETS) | tee /dev/stderr)"

fmt: $(TARGETS_FMT)
# @go fmt

$(TARGETS_FMT): fmt-%: %
	@gofmt -s -w $</

lint: $(GO_LINT)
	@$(GO_LINT) ./...

checkstyle: $(GO_LINT)
	@$(GO_LINT) --checkstyle ./... > checkstyle.xml

$(GO_BINDATA):
	go get -u github.com/jteeuwen/go-bindata/...

show_version:
	@echo ${VERSION_GIT}

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
