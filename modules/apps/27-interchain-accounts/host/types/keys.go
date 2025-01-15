package types

import (
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// SubModuleName defines the interchain accounts host module name
	SubModuleName = "icahost"

	// StoreKey is the store key string for the interchain accounts host module
	StoreKey = SubModuleName

	// ParamsKey is the key to use for the storing params.
	ParamsKey = "params"

	// AllowAllHostMsgs holds the string key that allows all message types on interchain accounts host module
	AllowAllHostMsgs = "*"
)

// ContainsMsgType returns true if the sdk.Msg TypeURL is present in allowMsgs, otherwise false
func ContainsMsgType(allowMsgs []string, msg sdk.Msg) bool {
	// check that wildcard * option for allowing all message types is the only string in the array, if so, return true
	if len(allowMsgs) == 1 && allowMsgs[0] == AllowAllHostMsgs {
		return true
	}

	return slices.Contains(allowMsgs, sdk.MsgTypeURL(msg))
}
