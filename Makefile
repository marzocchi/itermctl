bin/itermctl: Makefile cmd/itermctl/main.go $(shell find pkg -name "*.go" -or -name "*.proto")
	go build -race -o bin/itermctl cmd/itermctl/main.go
