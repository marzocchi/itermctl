.PHONY: test

bin/itermctl: Makefile cmd/itermctl/main.go $(shell find pkg -name "*.go" -or -name "*.proto")
	go build -race -o bin/itermctl cmd/itermctl/main.go

test_integration:
	go test -race -count=1 -v -tags test_with_iterm mrz.io/itermctl/pkg/...

test:
	go test -race -count=1 -v mrz.io/itermctl/pkg/...
