PLUGIN_NAME=helmfile

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

build: protoc protoc-gen-go
	go generate
	go build -o ./bin/waypoint-plugin-${PLUGIN_NAME} ./main.go

check_target:
ifndef TARGET
	$(error TARGET is undefined)
endif

install: check_target build
	mkdir -p $(TARGET)/.waypoint/plugins
	cp ./bin/waypoint-plugin-${PLUGIN_NAME} $(TARGET)/.waypoint/plugins/

.PHONY: test/monochart
test/monochart:
	TARGET=examples/monochart make install
	TARGET=examples/monochart make kubernetes-example
	cd examples/monochart; waypoint init
	cd examples/monochart; waypoint up

.PHONY: test
test: test/monochart

kubernetes-example: DIR=$(shell pwd)/$(TARGET)
kubernetes-example:
ifeq (, $(K8S_EXAMPLE))
	echo "Downloading kuberntes example"
	@{ \
	set -e ;\
	K8S_EXAMPLE_TMP_DIR=$$(mktemp -d) ;\
	cd $$K8S_EXAMPLE_TMP_DIR ;\
	git clone https://github.com/hashicorp/waypoint-examples.git ;\
	cp -r waypoint-examples/kubernetes/nodejs $(DIR)/ ;\
	rm -rf $$K8S_EXAMPLE_TMP_DIR ;\
	}
K8S_EXAMPLE=$(TARGET)/nodejs
endif

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	OS=linux
endif
ifeq ($(UNAME_S),Darwin)
	OS=osx
endif

protoc:
ifeq (, $(shell which protoc))
	echo "Downloading protoc"
	@{ \
	set -e ;\
	PROTOC_TMP_DIR=$$(mktemp -d) ;\
	cd $$PROTOC_TMP_DIR ;\
	curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v3.13.0/protoc-3.13.0-$(OS)-x86_64.zip ;\
	unzip protoc-3.13.0-$(OS)-x86_64.zip ;\
	cp bin/protoc $(GOBIN)/protoc ;\
	rm -rf $$PROTOC_TMP_DIR ;\
	}
PROTOC=$(GOBIN)/protoc
else
PROTOC=$(shell which protoc)
endif

protoc-gen-go:
ifeq (, $(shell which protoc-gen-go))
	echo "Downloading protoc plugins"
	@{ \
	set -e ;\
	PROTOC_TMP_DIR=$$(mktemp -d) ;\
	cd $$PROTOC_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/golang/protobuf/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc ;\
	rm -rf $$PROTOC_TMP_DIR ;\
	}
endif

goreleaser:
ifeq (, $(shell which goreleaser))
	echo "Downloading goreleaser"
	@{ \
	set -e ;\
	GORELEASER_TMP_DIR=$$(mktemp -d) ;\
	cd $$GORELEASER_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/goreleaser/goreleaser ;\
	rm -rf $$GORELEASER_TMP_DIR ;\
	}
endif

.PHONY: release
release:
	echo Please set GPG_FINGERPRINT
	gpg --armor --detach-sign
	goreleaser release --rm-dist

.PHONY: release/test
release/test: goreleaser
	goreleaser release --skip-publish --snapshot --rm-dist --skip-sign
