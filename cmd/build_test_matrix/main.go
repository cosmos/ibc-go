package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

const (
	testNamePrefix     = "Test"
	testFileNameSuffix = "_test.go"
	e2eTestDirectory   = "e2e"
	testEntryPointEnv  = "TEST_ENTRYPOINT"
	testExclusionsEnv  = "TEST_EXCLUSIONS"
	testNameEnv        = "TEST_NAME"
)

type GithubActionTestMatrix struct {
	Include []TestSuitePair `json:"include"`
}

type TestSuitePair struct {
	Suite string `json:"suite"`
}

func main() {
	githubActionMatrix, err := getGithubActionMatrixForTests(e2eTestDirectory, getExcludedTestFunctions())
	if err != nil {
		fmt.Printf("error generating github action json: %s", err)
		os.Exit(1)
	}

	ghBytes, err := json.Marshal(githubActionMatrix)
	if err != nil {
		fmt.Printf("error marshalling github action json: %s", err)
		os.Exit(1)
	}
	fmt.Println(string(ghBytes))
}

func getExcludedTestFunctions() []string {
	exclusions, ok := os.LookupEnv(testExclusionsEnv)
	if !ok {
		return nil
	}
	return strings.Split(exclusions, ",")
}

func getGithubActionMatrixForTests(e2eRootDirectory string, excludedItems []string) (GithubActionTestMatrix, error) {
	testSuites := map[string]bool{}
	fset := token.NewFileSet()
	err := filepath.Walk(e2eRootDirectory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking e2e directory: %s", err)
		}

		if !strings.HasSuffix(path, testFileNameSuffix) {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("failed parsing file: %s", err)
		}

		suiteName, err := extractSuiteName(f)
		if err != nil {
			return nil
		}

		if slices.Contains(excludedItems, suiteName) {
			return nil
		}

		testSuites[suiteName] = true
		return nil
	})
	if err != nil {
		return GithubActionTestMatrix{}, err
	}

	gh := GithubActionTestMatrix{Include: []TestSuitePair{}}
	for suiteName := range testSuites {
		gh.Include = append(gh.Include, TestSuitePair{Suite: suiteName})
	}

	sort.SliceStable(gh.Include, func(i, j int) bool {
		return gh.Include[i].Suite < gh.Include[j].Suite
	})

	if len(gh.Include) == 0 {
		return GithubActionTestMatrix{}, errors.New("no test suites found")
	}

	return gh, nil
}

func extractSuiteName(file *ast.File) (string, error) {
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			if isTestSuiteMethod(f) {
				return f.Name.Name, nil
			}
		}
	}
	return "", fmt.Errorf("no test suite found in file %s", file.Name.Name)
}

func isTestSuiteMethod(f *ast.FuncDecl) bool {
	return strings.HasPrefix(f.Name.Name, testNamePrefix) && len(f.Type.Params.List) == 1
}
