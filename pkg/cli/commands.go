package cli

import (
	"fmt"
	"os/exec"
	"strings"
)

func runKubectlCommand(input string) (string, error) {
	args := strings.Fields(input)
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	return string(output), nil
}
