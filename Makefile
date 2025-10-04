SOURCES = $(shell find cmd/resourceinjector -name '*.go')

PATH := $(shell pwd)/bin:$(PATH)

all: bin/kustomize-plugin-resourceinjector

bin/kustomize-plugin-resourceinjector: $(SOURCES)
	go build -o $@ ./cmd/resourceinjector

test: test/kustomize-plugin-resourceinjector

.PHONY: test/%
test/kustomize-plugin-%: bin/kustomize-plugin-%
	go test -v ./cmd/$*/...