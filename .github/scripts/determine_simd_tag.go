package main

import (
	"flag"
	"fmt"
	"os"
)

var prNum int
var ref string

func init() {
	flag.IntVar(&prNum, "pr", 0, "the number of the pr")
	flag.StringVar(&ref, "ref", "", "the github ref")
	flag.Parse()
}

// in the context of a GithubAction workflow, the PR is the event number. So if the ref is not specified
// but the event number is, that means we are running for a PR. If the ref is specified, this means
// we have merged the PR, so we want to use the ref as a tag instead of the PR number.
func main() {
	if prNum == 0 && ref == "" {
		fmt.Printf("must specify one or bot of [pr, ref]")
		os.Exit(1)
	}
	fmt.Printf(getSimdTag(prNum, ref))
}

func getSimdTag(prNum int, ref string) string {
	if ref != "" {
		return ref
	}
	return fmt.Sprintf("pr-%d", prNum)
}
