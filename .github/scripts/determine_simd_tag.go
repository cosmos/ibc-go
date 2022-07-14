package main

import (
	"flag"
	"fmt"
)

var prNum string

func init() {
	flag.StringVar(&prNum, "pr", "", "the number of the pr")
	flag.Parse()
}

// in the context of a GithubAction workflow, the PR is non empty if it is a pr. When
// code is merged to main, it will be empty. In this case we just use the "main" tag.
func main() {
	fmt.Printf(getSimdTag(prNum))
}

func getSimdTag(prNum string) string {
	if prNum == "" {
		return "main"
	}
	return fmt.Sprintf("pr-%s", prNum)
}
