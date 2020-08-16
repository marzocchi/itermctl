package applescript

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func RunScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-")
	output := &bytes.Buffer{}
	cmd.Stdin = strings.NewReader(script)
	cmd.Stdout = output
	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("applescript: %w", err)
	}
	return output.String(), nil
}

func IsRunning(appName string) (bool, error) {
	out, err := RunScript(fmt.Sprintf("return application %q is running", appName))

	if err != nil {
		return false, fmt.Errorf("applescript: could not determine %q running state: %w", appName, err)
	}

	if "t" == out[0:1] {
		return true, nil
	}

	return false, nil
}

