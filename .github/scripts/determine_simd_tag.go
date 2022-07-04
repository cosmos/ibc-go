package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Println("must specify exactly 2 args, ref and PR number")
		os.Exit(1)
	}

	tag, err := determineSimdTag(args[1], args[2])
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
func determineSimdTag(ref, prNumber string) (string, error) {
	if ref != "" {
		return ref, nil
	}
	prNumm, err := strconv.Atoi(prNumber)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pr-%d", prNumm), nil
}
