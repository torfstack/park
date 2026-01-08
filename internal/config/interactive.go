package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	inputFile = os.Stdin
)

func guidedInitialization(config *Config) error {
	scanner := bufio.NewScanner(inputFile)

	input, err := ask(scanner, fmt.Sprintf("Enter local directory path [default: %s]", config.LocalDir))
	if err != nil {
		return err
	}
	if input != "" {
		config.LocalDir = input
	}

	input, err = ask(scanner, fmt.Sprintf("Enter sync interval (e.g. 30s, 1m) [default: %s]", config.SyncInterval))
	if err != nil {
		return err
	}
	if input != "" {
		duration, err := time.ParseDuration(input)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
		config.SyncInterval = duration
	}

	return nil
}

func ask(scanner *bufio.Scanner, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("could not read user input: %w", err)
		}
		return "", nil // EOF or closed input
	}
	return strings.TrimSpace(scanner.Text()), nil
}
