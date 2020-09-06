.PHONY: integration_test prepare_environment

ITERM_DIR="$(HOME)/Library/Application Support/iTerm2"

bin/itermctl: Makefile cmd/itermctl/main.go $(shell find pkg -name "*.go" -or -name "*.proto")
	go build -race -o bin/itermctl cmd/itermctl/main.go

prepare_environment:
	mkdir $(ITERM_DIR) || true
	mkdir $(ITERM_DIR)/DynamicProfiles || true

	./scripts/disable-auth.py $(ITERM_DIR)/disable-automation-auth
	sudo chown root $(ITERM_DIR)/disable-automation-auth

	curl https://iterm2.com/shell_integration/zsh > /tmp/iterm2-shell-integration-for-itermctl-test.zsh

	defaults write com.googlecode.iterm2 EnableAPIServer 1
	defaults write com.googlecode.iterm2 NeverWarnAboutShortLivedSessions_0365D7C9-AD5C-4957-8FFF-DB296B96EF0C 1

	./scripts/gen-profile.sh "$(shell pwd)/scripts/zsh-with-iterm-integration.sh /tmp/iterm2-shell-integration-for-itermctl-test.zsh" \
		> $(ITERM_DIR)/DynamicProfiles/itermctl-test-profile.json

	open /Applications/iTerm.app

integration_test: prepare_environment
	go test -race -count=1 -v -tags test_with_iterm mrz.io/itermctl/pkg/...
