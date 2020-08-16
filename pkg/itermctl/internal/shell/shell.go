package shell

import (
	"bytes"
	"fmt"
	"os/exec"
)

func Which(command string) (string, error) {
	which := exec.Command("/usr/bin/Which", command)
	buffer := &bytes.Buffer{}

	which.Stdout = buffer
	err := which.Run()

	if err != nil {
		return "", fmt.Errorf("get command path: %w", err)
	}

	return string(buffer.Bytes()[:len(buffer.Bytes())-1]), nil
}
