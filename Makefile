.PHONY: integration_test prepare_environment prepare_profile get_proto

ITERM_DIR="$(HOME)/Library/Application Support/iTerm2"

bin/itermctl: Makefile cmd/itermctl/main.go $(shell find pkg -name "*.go" -or -name "*.proto")
	go build -race -o bin/itermctl cmd/itermctl/main.go

prepare_environment:
	mkdir $(ITERM_DIR) || true
	mkdir $(ITERM_DIR)/DynamicProfiles || true

	./scripts/disable-auth.py $(ITERM_DIR)/disable-automation-auth
	sudo chown root $(ITERM_DIR)/disable-automation-auth

	defaults write com.googlecode.iterm2 EnableAPIServer 1

	open /Applications/iTerm.app

prepare_profile:
	curl https://iterm2.com/shell_integration/zsh > /tmp/iterm2-shell-integration-for-itermctl-test.zsh
	defaults write com.googlecode.iterm2 NeverWarnAboutShortLivedSessions_0365D7C9-AD5C-4957-8FFF-DB296B96EF0C 1

	./scripts/gen-profile.sh "$(shell pwd)/scripts/zsh-with-iterm-integration.sh /tmp/iterm2-shell-integration-for-itermctl-test.zsh" \
		> $(ITERM_DIR)/DynamicProfiles/itermctl-test-profile.json

integration_test: prepare_environment prepare_profile
	go test -race -count=1 -v -tags test_with_iterm mrz.io/itermctl/pkg/...

get_proto:
	rm pkg/itermctl/proto/api.proto || true
	curl -L https://raw.githubusercontent.com/gnachman/iTerm2/master/proto/api.proto > pkg/itermctl/proto/api.proto

pkg/itermctl/proto/api.pb.go: get_proto pkg/itermctl/proto/api.proto
	rm pkg/itermctl/proto/api.pb.go || true
	protoc --go_out=. pkg/itermctl/proto/api.proto

