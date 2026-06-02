package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// ghToken returns the token the GitHub CLI is currently authenticated with,
// i.e. the output of `gh auth token`. It lets Setzer reuse an existing gh login
// instead of a separately-pasted PAT.
func ghToken() (string, error) {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("gh returned an empty token (is `gh auth login` done?)")
	}
	return token, nil
}
