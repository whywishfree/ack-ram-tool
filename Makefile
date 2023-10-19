
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
VERSION ?= $(shell git describe --tags --abbrev=0)
CGO_ENABLED ?= 0
LDFLAGS := -extldflags "-static"
LDFLAGS += -X github.com/AliyunContainerService/ack-ram-tool/pkg/version.Version=$(VERSION)
LDFLAGS += -X github.com/AliyunContainerService/ack-ram-tool/pkg/version.GitCommit=$(GIT_COMMIT)

CLUSTER ?= ''
CLUSTER_ID ?= $(CLUSTER)
cid ?= $(CLUSTER_ID)

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -a -o ack-ram-tool \
	cmd/ack-ram-tool/main.go

.PHONY: test
test:
	go test -race -v ./...
	cd pkg/credentials/provider && go test -race -v ./...

.PHONY: e2e
e2e:
	bash ./examples/rrsa/e2e-test/e2e.sh $(cid)
	bash ./examples/credential-plugin/e2e.sh $(cid)

.PHONY: lint
lint: deps fmt vet

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: deps
deps:
	go mod tidy && go mod vendor
