package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EmptyMsgs is an empty implementation of the SDK Handler interface
func EmptyMsgs() []sdk.Msg {
	return []sdk.Msg{}
}

// EmptyRoute is an empty implementation of the SDK Route interface
func EmptyRoute() string {
	return ""
}
