package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	testSuitePrefix    = "Test"
	testSuiteSuffix    = "Suite"
	testFileNameSuffix = "_test.go"
	e2eTestDirectory   = "e2e"
	// testExclusionsEnv is a comma separated list of test function names that will not be included
	// in the results of this script.
	testSuiteExclusionsEnv = "TEST_SUITE_EXCLUSIONS"
	// testNameEnv if provided returns a single test entry so that only one test is actually run.
)

// GithubActionTestMatrix represents
type GithubActionTestMatrix struct {
	Include []TestSuite `json:"include"`
}

type TestSuite struct {
	EntryPoint string `json:"entrypoint"`
}

func main() {
	githubActionMatrix, err := getGithubActionMatrixForTests(e2eTestDirectory, getExcludedTestSuiteFunctions())
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

// getExcludedTestFunctions returns a list of test functions that we don't want to run.
func getExcludedTestSuiteFunctions() []string {
	exclusions, ok := os.LookupEnv(testSuiteExclusionsEnv)
	if !ok {
		return nil
	}
	return strings.Split(exclusions, ",")
}

func contains(s string, items []string) bool {
	for _, elem := range items {
		if elem == s {
			return true
		}
	}
	return false
}

// getGithubActionMatrixForTests returns a json string representing the contents that should go in the matrix
// field in a github action workflow. This string can be used with `fromJSON(str)` to dynamically build
// the workflow matrix to include all E2E tests under the e2eRootDirectory directory.
func getGithubActionMatrixForTests(e2eRootDirectory string, excludedItems []string) (GithubActionTestMatrix, error) {
	testSuite := []string{}
	fset := token.NewFileSet()
	err := filepath.Walk(e2eRootDirectory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking e2e directory: %s", err)
		}

		// only look at test files
		if !strings.HasSuffix(path, testFileNameSuffix) {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("failed parsing file: %s", err)
		}

		suiteNameForFile, err := extractSuite(f)
		if err != nil {
			return fmt.Errorf("failed extracting test suite name: %s", err)
		}

		if contains(suiteNameForFile, excludedItems) {
			return nil
		}

		if suiteNameForFile != "" {
			testSuite = append(testSuite, suiteNameForFile)
		}

		return nil
	})
	if err != nil {
		return GithubActionTestMatrix{}, err
	}

	gh := GithubActionTestMatrix{
		Include: []TestSuite{},
	}

	for _, testSuiteName := range testSuite {
		gh.Include = append(gh.Include, TestSuite{
			EntryPoint: testSuiteName,
		})
	}

	return gh, nil
}

// extractSuiteAndTestNames extracts the name of the test suite function as well
// as all tests associated with it in the same file.
func extractSuite(file *ast.File) (string, error) {
	var suiteNameForFile string

	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			functionName := f.Name.Name
			if isTestSuiteMethod(f) {
				if suiteNameForFile != "" {
					return "", fmt.Errorf("found a second test function: %s when %s was already found", f.Name.Name, suiteNameForFile)
				}
				suiteNameForFile = functionName
				continue
			}
		}
	}
	return suiteNameForFile, nil
}

// isTestSuiteMethod returns true if the function is a test suite function.
// e.g. func TestFeeMiddlewareTestSuite(t *testing.T) { ... }
func isTestSuiteMethod(f *ast.FuncDecl) bool {
	return strings.HasPrefix(f.Name.Name, testSuitePrefix) && strings.HasSuffix(f.Name.Name, testSuiteSuffix)
}
