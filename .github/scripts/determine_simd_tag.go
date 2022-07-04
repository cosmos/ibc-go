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
