package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

// OnRecvPacket processes a GMP packet.
//
// If the sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	data *types.GMPPacketData,
	sourcePort,
	sourceChannel,
	destPort,
	destChannel string,
) error {
	panic("not implemented") // TODO: Implement
}
