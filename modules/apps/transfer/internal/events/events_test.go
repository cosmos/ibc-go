// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
)

// TestEmitOnRecvPacketEventAckSuccess checks the ack success attribute, including
// for a nil acknowledgement, which is what the deferred call in OnRecvPacket passes
// when the receive logic panics before an acknowledgement is assigned.
func TestEmitOnRecvPacketEventAckSuccess(t *testing.T) {
	testCases := []struct {
		name       string
		ack        ibcexported.Acknowledgement
		expSuccess string
	}{
		{"success ack", channeltypes.NewResultAcknowledgement([]byte{byte(1)}), "true"},
		{"error ack", channeltypes.NewErrorAcknowledgement(types.ErrInvalidAmount), "false"},
		{"nil ack", nil, "false"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.NewContext(nil, cmtproto.Header{}, false, nil).WithEventManager(sdk.NewEventManager())
			data := types.NewInternalTransferRepresentation(
				types.Token{Denom: types.NewDenom("atom"), Amount: "100"},
				"sender", "receiver", "",
			)

			require.NotPanics(t, func() {
				events.EmitOnRecvPacketEvent(ctx, data, tc.ack, nil)
			})

			var found bool
			for _, event := range ctx.EventManager().Events() {
				if event.Type != types.EventTypePacket {
					continue
				}
				attr, ok := event.GetAttribute(types.AttributeKeyAckSuccess)
				require.True(t, ok)
				require.Equal(t, tc.expSuccess, attr.Value)
				found = true
			}
			require.True(t, found)
		})
	}
}
