SOURCES = $(shell find cmd/resourceinjector -name '*.go')

all: bin/resourceinjector

bin/resourceinjector: $(SOURCES)
	go build -o $@ ./cmd/resourceinjector

test: test/resourceinjector

.PHONY: test/%
test/%: bin/%
	go test -v ./cmd/$*/...