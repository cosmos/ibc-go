package ibctesting_test

import (
	"testing"

	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// TODO: Remove this before merging.
func TestSetupTestingApp(t *testing.T) {
	app, genState := ibctesting.SetupTestingApp()

	_, _ = app, genState
}
