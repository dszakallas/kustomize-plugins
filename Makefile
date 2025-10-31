SOURCES = $(shell find cmd/resourceinjector -name '*.go')

PATH := $(shell pwd)/bin:$(PATH)

all: \
  bin/kustomize-plugin-resourceinjector \
  bin/kustomize-plugin-yqtransform

bin/kustomize-plugin-%: $(SOURCES)
	go build -o $@ ./cmd/$*

test: \
	test/kustomize-plugin-resourceinjector \
	test/kustomize-plugin-yqtransform

.PHONY: test/%
test/kustomize-plugin-%: bin/kustomize-plugin-%
	go test -v ./cmd/$*/...