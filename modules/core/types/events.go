package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const ErrorAttributeKeySuffix = "-error"

// ConvertToErrorEvents converts all events to error events by appending the
// error attribute suffix to each event's attribute key.
func ConvertToErrorEvents(events sdk.Events) sdk.Events {
	if events == nil {
		return nil
	}

	newEvents := make(sdk.Events, len(events))
	for i, event := range events {
		newAttributes := make([]sdk.Attribute, len(event.Attributes))
		for j, attribute := range event.Attributes {
			newAttributes[j] = sdk.NewAttribute(attribute.Key+ErrorAttributeKeySuffix, attribute.Value)
		}

		// no need to append the error attribute suffix to the event type because
		// the event type is not associated to a value that can be misinterpreted
		newEvents[i] = sdk.NewEvent(event.Type, newAttributes...)
	}

	return newEvents
}
