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

func main() {
	if prNum == 0 && ref == "" {
		fmt.Printf("must specify exactly one of [pr, ref]")
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
