package testsuite

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"
)

// ABCIToSDKEvents converts a list of ABCI events to Cosmos SDK events.
func ABCIToSDKEvents(abciEvents []abci.Event) sdk.Events {
	var events sdk.Events
	for _, evt := range abciEvents {
		var attributes []sdk.Attribute
		for _, attr := range evt.GetAttributes() {
			attributes = append(attributes, sdk.NewAttribute(attr.Key, attr.Value))
		}

		events = events.AppendEvent(sdk.NewEvent(evt.GetType(), attributes...))
	}

	return events
}
