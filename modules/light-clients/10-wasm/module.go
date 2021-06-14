package wasm

import (
	"github.com/cosmos/ibc-go/modules/light-clients/10-wasm/types"
)

// Name returns the IBC client name
func Name() string {
	return types.SubModuleName
}
