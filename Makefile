.PHONY: integration_test prepare_environment prepare_profile update_proto

ITERM_DIR="$(HOME)/Library/Application Support/iTerm2"

integration_test: update_proto prepare_environment prepare_profile
	go test -race -count=1 -v -tags test_with_iterm mrz.io/itermctl/...

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

update_proto:
	rm proto/api.proto || true
	curl -L https://raw.githubusercontent.com/gnachman/iTerm2/master/proto/api.proto > iterm2/api.proto
	go generate

