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
	Pairs []Pair `json:"include"`
}

type Pair struct {
	Test  string `json:"test"`
	Suite string `json:"suite"`
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
		Pairs: []Pair{},
		//Test:  []string{},
		//Suite: []string{},
	}

	for testSuiteName, testCases := range testSuiteMapping {
		for _, testCaseName := range testCases {
			gh.Pairs = append(gh.Pairs, Pair{
				Test:  testCaseName,
				Suite: testSuiteName,
			})
			//gh.Test = append(gh.Test, testCaseName)
			//gh.Suite = append(gh.Suite, testSuiteName)
		}
	}

	ghBytes, err := json.Marshal(gh)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(ghBytes))
}
