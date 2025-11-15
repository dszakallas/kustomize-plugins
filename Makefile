GO_FILES = $(shell find . -name '*.go' -not -name '*_test.go') go.mod go.sum

all: \
  bin/kustomize-plugin-resourceinjector \
  bin/kustomize-plugin-yqtransform

bin/kustomize-plugin-%: $(GO_FILES)
	go build -o $@ ./cmd/$*

test: \
	test/kustomize-plugin-resourceinjector \
	test/kustomize-plugin-yqtransform

.PHONY: test/%
test/kustomize-plugin-%: bin/kustomize-plugin-%
	go test -v ./test/$*/...