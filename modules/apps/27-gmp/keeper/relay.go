package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

// OnRecvPacket processes a GMP packet.
// Returns the data result of the execution if successful.
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	data *types.GMPPacketData,
	sourcePort,
	sourceClient,
	destPort,
	destClient string,
) ([]byte, error) {
	accountId := types.NewAccountIdentifier(destClient, data.Sender, data.Salt)

	_, err := k.getOrCreateICS27Account(ctx, &accountId)
	if err != nil {
		return nil, err
	}

	panic("not implemented") // TODO: Implement
}
