package mock

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
)

var _ api.IBCModule = (*IBCModule)(nil)

type IBCModule struct{}

func (IBCModule) OnSendPacket(ctx context.Context, sourceID string, destinationID string, sequence uint64, data channeltypesv2.PacketData, signer sdk.AccAddress) error {
	panic("implement me")
}
