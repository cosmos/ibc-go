package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/types"
)

func ConvertToErrorEvents(events sdk.Events) sdk.Events {
	newEvents := make(sdk.Events, len(events))
	for i, event := range events {
		newAttributes := make([]sdk.Attribute, len(event.Attributes))
		for j, attribute := range event.Attributes {
			newAttributes[j] = sdk.NewAttribute(attribute.Key+types.ErrorAttributeKeySuffix, attribute.Value)
		}

		newEvents[i] = sdk.NewEvent(event.Type, newAttributes...)
	}

	return newEvents
}
