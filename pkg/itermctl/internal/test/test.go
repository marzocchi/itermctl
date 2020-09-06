package test

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

const ShellIntegrationUrl = "https://iterm2.com/shell_integration/zsh"

func AppName(t *testing.T) string {
	return fmt.Sprintf("itermctl_%s", t.Name())
}

func SaveShellIntegrationPluginForZsh() (string, error) {
	client := http.DefaultClient
	client.Timeout = 2 * time.Second

	resp, err := client.Get(ShellIntegrationUrl)
	if err != nil {
		return "", fmt.Errorf("get shell integration: %w", err)
	}

	f, err := ioutil.TempFile("", "iterm-shell-integration-for-zsh-*")
	if err != nil {
		return "", fmt.Errorf("get shell integration: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("get shell integration: %w", err)
	}

	if err := f.Sync(); err != nil {
		return "", fmt.Errorf("get shell integration: %w", err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("get shell integration: %w", err)
	}

	logrus.Errorf("saved at: %s", f.Name())

	return f.Name(), nil
}
