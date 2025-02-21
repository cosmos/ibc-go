package errors

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	coretypes "github.com/cosmos/ibc-go/v10/modules/core/types"
)

// ConvertToErrorEvents converts all events to error events by appending the
// error attribute prefix to each event's attribute key.
func ConvertToErrorEvents(events sdk.Events) sdk.Events {
	if events == nil {
		return nil
	}

	newEvents := make(sdk.Events, len(events))
	for i, event := range events {
		newEvents[i] = sdk.NewEvent(coretypes.ErrorAttributeKeyPrefix + event.Type)
		for _, attribute := range event.Attributes {
			newEvents[i] = newEvents[i].AppendAttributes(sdk.NewAttribute(coretypes.ErrorAttributeKeyPrefix+attribute.Key, attribute.Value))
		}
	}

	return newEvents
}
