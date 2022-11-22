package dymint

import (
	"github.com/cosmos/ibc-go/v3/modules/light-clients/01-dymint/types"
)

// Name returns the IBC client name
func Name() string {
	return types.SubModuleName
}
