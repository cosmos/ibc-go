package mock

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

type IBCApp struct {
	OnSendPacket func(ctx context.Context, sourceID string, destinationID string, sequence uint64, data channeltypesv2.PacketData, signer sdk.AccAddress) error
	OnRecvPacket func(ctx context.Context, sourceID string, destinationID string, data channeltypesv2.PacketData, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult
}
