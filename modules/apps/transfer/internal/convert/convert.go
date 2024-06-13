package convert

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// PacketDataV1ToV2 converts a v1 packet data to a v2 packet data. The packet data is validated
// before conversion.
func PacketDataV1ToV2(packetData types.FungibleTokenPacketData) (types.FungibleTokenPacketDataV2, error) {
	if err := packetData.ValidateBasic(); err != nil {
		return types.FungibleTokenPacketDataV2{}, errorsmod.Wrapf(err, "invalid packet data")
	}

	denom := types.ExtractDenomFromPath(packetData.Denom)
	return types.FungibleTokenPacketDataV2{
		Tokens: []types.Token{
			{
				Denom:  denom,
				Amount: packetData.Amount,
			},
		},
		Sender:     packetData.Sender,
		Receiver:   packetData.Receiver,
		Memo:       packetData.Memo,
		Forwarding: nil,
	}, nil
}
