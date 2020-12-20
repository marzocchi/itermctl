package auth

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"mrz.io/itermctl/internal/applescript"
	"os"
	"strings"
	"syscall"
)

var (
	disableAuthFile = "~/Library/Application Support/iTerm2/disable-automation-auth"
	magicString     = "61DF88DC-3423-4823-B725-22570E01C027"
)

// Disabled checks if iTerm2 is configured to accept connections from every client, or if a client should first
// request the cookie and key instead. If auth is for sure disabled it returns nil, otherwise it returns an error
// with a description of why auth appears enabled or it was not possible to complete detection.
// See https://iterm2.com/python-api-auth.html for documentation of iTerm2's API Security.
func Disabled() error {
	disableAuthFilePath, err := homedir.Expand(disableAuthFile)

	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	info, err := os.Stat(disableAuthFilePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("auth: %s does not exist", disableAuthFilePath)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("auth: %s exists, but is not a regular file", disableAuthFilePath)
	}

	if info.Sys().(*syscall.Stat_t).Uid != 0 {
		return fmt.Errorf("auth: %s exists, but is not owned by root", disableAuthFilePath)
	}

	f, err := os.Open(disableAuthFilePath)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	magicString, err := MagicString()
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	scanner := bufio.NewScanner(f)
	scanner.Scan()

	if scanner.Text() != magicString {
		return fmt.Errorf("auth: contents of %s do not match expected", disableAuthFile)
	}

	return nil
}

// MagicString returns the expected contents of the `disable-automation-auth` file.
// See https://iterm2.com/python-api-auth.html for documentation of iTerm2's API Security.
func MagicString() (string, error) {
	disableAuthFilePath, err := homedir.Expand(disableAuthFile)
	if err != nil {
		return "", fmt.Errorf("auth: %w", err)
	}

	encodedAuthFilePath := hex.EncodeToString([]byte(disableAuthFilePath))
	return encodedAuthFilePath + " " + magicString, nil
}

// RequestCookieAndKey requests the cookie and key to authenticate with iTerm2 via Applescript, potentially triggering
// iTerm2's or macOS confirmation dialogs. If activate is true, iTerm2 will be started automatically if it's currently
// not running. If iTerm2 is not running and and active is false, an error will be returned without attempting to
// request the cookie and key.
// See https://iterm2.com/python-api-auth.html for documentation of iTerm2's API Security.
func RequestCookieAndKey(appName string, activate bool) (string, string, error) {
	var activateCommand string

	if activate {
		activateCommand = "activate"
	} else {
		running, err := applescript.IsRunning("iTerm2")
		if err != nil {
			return "", "", fmt.Errorf("request cookie and key: %w", err)
		}

		if !running {
			return "", "", fmt.Errorf("request cookie and key: iTerm2 is not running and activation is disabled")
		}
	}

	script := fmt.Sprintf(`
		tell app "iTerm"
			%s
			request cookie and key for app named %q
		end
	`, activateCommand, appName)

	out, err := applescript.RunScript(script)
	if err != nil {
		return "", "", fmt.Errorf("request cookie and key: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(out), " ")

	return parts[0], parts[1], nil
}
