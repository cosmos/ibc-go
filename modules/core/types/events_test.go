package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/types"
)

func TestConvertToErrorEvents(t *testing.T) {
	var (
		events    sdk.Events
		expEvents sdk.Events
	)

	tc := []struct {
		name     string
		malleate func()
	}{
		{
			"success: nil events",
			func() {
				events = nil
				expEvents = nil
			},
		},
		{
			"success: empty events",
			func() {
				events = sdk.Events{}
				expEvents = sdk.Events{}
			},
		},
		{
			"success: event with no attributes",
			func() {
				events = sdk.Events{
					sdk.NewEvent("testevent"),
				}
				expEvents = sdk.Events{
					sdk.NewEvent("testevent"),
				}
			},
		},
		{
			"success: event with attributes",
			func() {
				events = sdk.Events{
					sdk.NewEvent("testevent",
						sdk.NewAttribute("key1", "value1"),
						sdk.NewAttribute("key2", "value2"),
					),
				}
				expEvents = sdk.Events{
					sdk.NewEvent("testevent",
						sdk.NewAttribute("key1"+types.ErrorAttributeKeySuffix, "value1"),
						sdk.NewAttribute("key2"+types.ErrorAttributeKeySuffix, "value2"),
					),
				}
			},
		},
		{
			"success: multiple events with attributes",
			func() {
				events = sdk.Events{
					sdk.NewEvent("testevent1",
						sdk.NewAttribute("key1", "value1"),
						sdk.NewAttribute("key2", "value2"),
					),
					sdk.NewEvent("testevent2",
						sdk.NewAttribute("key3", "value3"),
						sdk.NewAttribute("key4", "value4"),
					),
				}
				expEvents = sdk.Events{
					sdk.NewEvent("testevent1",
						sdk.NewAttribute("key1"+types.ErrorAttributeKeySuffix, "value1"),
						sdk.NewAttribute("key2"+types.ErrorAttributeKeySuffix, "value2"),
					),
					sdk.NewEvent("testevent2",
						sdk.NewAttribute("key3"+types.ErrorAttributeKeySuffix, "value3"),
						sdk.NewAttribute("key4"+types.ErrorAttributeKeySuffix, "value4"),
					),
				}
			},
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			// initial events and expected events are reset so that the test fails if
			// the malleate function does not set them
			events = nil
			expEvents = sdk.Events{}

			tc.malleate()

			newEvents := types.ConvertToErrorEvents(events)
			require.Equal(t, expEvents, newEvents)
		})
	}
}
