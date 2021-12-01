package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the interchain accounts host module name
	ModuleName = "icahost"

	// StoreKey is the store key string for the interchain accounts host module
	StoreKey = ModuleName
)

// ContainsType returns true if the sdk.Msg TypeURL is present in allowMsgs, otherwise false
func ContainsType(allowMsgs []string, msg sdk.Msg) bool {
	for _, v := range allowMsgs {
		if v == sdk.MsgTypeURL(msg) {
			return true
		}
	}

	return false
}
