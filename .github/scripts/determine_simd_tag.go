package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	args := os.Args

	if len(args) < 2 {
		fmt.Printf("a ref or PR number is expected, provided: %+v\n", args)
		os.Exit(1)
	}

	tag, err := determineSimdTag(args[1])
	if err != nil {
		fmt.Printf("failed to determine tag: %s", err)
		os.Exit(1)
	}
	fmt.Println(tag)
}

// determineSimdTag returns the tag which should be used for the E2E test image.
// when a ref is specified, this will usually be "main" which is the tag that should be
// used once a branch has been merged to main. If a PR number is specified, then the format
// of the tag will be "pr-1234".
func determineSimdTag(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("empty input was provided")
	}

	// attempt to extract PR number
	prNumm, err := strconv.Atoi(input)
	if err == nil {
		return fmt.Sprintf("pr-%d", prNumm), nil
	}

	// a ref was provided instead, e.g. "main"
	return input, nil
}
