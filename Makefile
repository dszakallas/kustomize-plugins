all: bin/resourceinjector

bin/%:
	go build -o $@ ./cmd/$*

test: test/resourceinjector

.PHONY: test/%
test/%: bin/%
	go test -v ./cmd/$*/...