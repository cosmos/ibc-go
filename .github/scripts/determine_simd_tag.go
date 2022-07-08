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

// in the context of a GithubAction workflow, the PR is the event number. So if the ref is not specified
// but the event number is, that means we are running for a PR. If the ref is specified, this means
// we have merged the PR, so we want to use the ref as a tag instead of the PR number.
func main() {
	fmt.Printf(getSimdTag(prNum))
}

func getSimdTag(prNum string) string {
	if prNum == "" {
		return "main"
	}
	return fmt.Sprintf("pr-%s", prNum)
}
