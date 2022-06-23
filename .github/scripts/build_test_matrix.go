package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

const (
	testNamePrefix = "Test"
)

// isTestSuiteMethod returns true if the function is a test suite function.
// e.g. func TestFeeMiddlewareTestSuite(t *testing.T) { ... }
func isTestSuiteMethod(f *ast.FuncDecl) bool {
	return strings.HasPrefix(f.Name.Name, testNamePrefix) && len(f.Type.Params.List) == 1
}

func isTestFunction(f *ast.FuncDecl) bool {
	return strings.HasPrefix(f.Name.Name, testNamePrefix) && len(f.Type.Params.List) == 0
}

type GithubActionTestMatrix struct {
	Includes []TestCaseSuitePair `json:"include"`
}

type TestCaseSuitePair struct {
	TestCase string `json:"testCase"`
	Suite    string `json:"suite"`
}

func main() {

	testSuiteMapping := map[string][]string{}

	fset := token.NewFileSet()
	err := filepath.Walk("e2e", func(path string, info fs.FileInfo, err error) error {
		// only look at test files
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			log.Panic(err)
		}

		var suiteNameForFile string
		var testCases []string

		for _, d := range f.Decls {
			if f, ok := d.(*ast.FuncDecl); ok {
				functionName := f.Name.Name
				if isTestSuiteMethod(f) {
					suiteNameForFile = functionName
					continue
				}
				if isTestFunction(f) {
					testCases = append(testCases, functionName)
				}
			}
		}

		if suiteNameForFile == "" {
			panic(fmt.Sprintf("file %s had no test suite test case", path))
		}

		testSuiteMapping[suiteNameForFile] = testCases

		return nil
	})

	if err != nil {
		panic(err)
	}

	gh := GithubActionTestMatrix{
		Includes: []TestCaseSuitePair{},
	}
	for testSuiteName, testCases := range testSuiteMapping {
		for _, testCaseName := range testCases {
			gh.Includes = append(gh.Includes, TestCaseSuitePair{
				TestCase: testCaseName,
				Suite:    testSuiteName,
			})
		}
	}

	ghBytes, err := json.MarshalIndent(gh, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(ghBytes))
}
