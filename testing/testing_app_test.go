package ibctesting_test

import (
	"testing"

	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/stretchr/testify/require"
)

// TODO: Remove this before merging.
func TestSetupTestingApp(t *testing.T) {
	app, genState := ibctesting.SetupTestingApp()

	router := app.GetIBCKeeper().PortKeeper.Router
	ok := router.HasRoute("mock")
	require.True(t, ok)

	_, _ = app, genState
}
