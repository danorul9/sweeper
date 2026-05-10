package appindex

import (
	"os/exec"
	"strings"
)

func BrewCaskList() ([]string, error) {
	cmd := exec.Command("brew", "list", "--cask")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	names := strings.Fields(string(output))
	return names, nil
}

func BrewList() ([]string, error) {
	cmd := exec.Command("brew", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	names := strings.Fields(string(output))
	return names, nil
}
