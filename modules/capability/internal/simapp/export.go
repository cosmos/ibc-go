package simapp

import (
	"errors"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// ExportAppStateAndValidators implements the runtime.AppI interface.
func (app *SimApp) ExportAppStateAndValidators(
	forZeroHeight bool, jailAllowedAddrs []string, modulesToExport []string,
) (servertypes.ExportedApp, error) {
	return servertypes.ExportedApp{}, errors.New("unsupported")
}
