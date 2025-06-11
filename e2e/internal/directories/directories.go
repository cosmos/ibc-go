package directories

import (
	"fmt"
	"os"
	"path"
	"strings"
)

const (
	e2eDir = "e2e"

	// DefaultGenesisExportPath is the default path to which Genesis debug files will be exported to.
	DefaultGenesisExportPath = "diagnostics/genesis.json"
)

// E2E finds the e2e directory above the test.
func E2E() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	const maxAttempts = 100
	count := 0
	for ; !strings.HasSuffix(wd, e2eDir) && count < maxAttempts; wd = path.Dir(wd) {
		count++
	}

	// arbitrary value to avoid getting stuck in an infinite loop if this is called
	// in a context where the e2e directory does not exist.
	if count == maxAttempts {
		return "", fmt.Errorf("unable to find e2e directory after %d tries", maxAttempts)
	}

	return wd, nil
}
