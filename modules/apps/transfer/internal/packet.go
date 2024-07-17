package internal

import (
	"encoding/json"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec/unknownproto"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes into a FungibleTokenPacketDataV2.
// The version of ics20 should be provided and should be either ics20-1 or ics20-2.
func UnmarshalPacketData(bz []byte, ics20Version string) (types.FungibleTokenPacketDataV2, error) {
	switch ics20Version {
	case types.V1:
		var datav1 types.FungibleTokenPacketData
		if err := json.Unmarshal(bz, &datav1); err != nil {
			return types.FungibleTokenPacketDataV2{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS20-V1 transfer packet data: %s", err.Error())
		}

		return packetDataV1ToV2(datav1)
	case types.V2:
		var datav2 types.FungibleTokenPacketDataV2
		if err := unknownproto.RejectUnknownFieldsStrict(bz, &datav2, unknownproto.DefaultAnyResolver{}); err != nil {
			return types.FungibleTokenPacketDataV2{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS20-V2 transfer packet data: %s", err.Error())
		}

		if err := proto.Unmarshal(bz, &datav2); err != nil {
			return types.FungibleTokenPacketDataV2{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS20-V2 transfer packet data: %s", err.Error())
		}

		if err := datav2.ValidateBasic(); err != nil {
			return types.FungibleTokenPacketDataV2{}, err
		}

		return datav2, nil
	default:
		return types.FungibleTokenPacketDataV2{}, errorsmod.Wrap(types.ErrInvalidVersion, ics20Version)
	}
}

// packetDataV1ToV2 converts a v1 packet data to a v2 packet data. The packet data is validated
// before conversion.
func packetDataV1ToV2(packetData types.FungibleTokenPacketData) (types.FungibleTokenPacketDataV2, error) {
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
		Forwarding: types.ForwardingPacketData{},
	}, nil
}
