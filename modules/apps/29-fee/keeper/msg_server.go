package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

var _ types.MsgServer = Keeper{}

// RegisterCounterpartyAddress is called by the relayer on each channelEnd and allows them to specify their counterparty address before relaying
// This ensures they will be properly compensated for forward relaying on the source chain since the destination chain must send back relayer's source address (counterparty address) in acknowledgement
// This function may be called more than once by relayers, in which case, the previous counterparty address will be overwritten by the new counterparty address
func (k Keeper) RegisterCounterpartyAddress(goCtx context.Context, msg *types.MsgRegisterCounterpartyAddress) (*types.MsgRegisterCounterpartyAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	counterpartyAddress, err := sdk.AccAddressFromBech32(msg.CounterpartyAddress)
	if err != nil {
		return &types.MsgRegisterCounterpartyAddressResponse{}, err
	}

	k.SetCounterpartyAddress(
		ctx, msg.Address, counterpartyAddress,
	)

	k.Logger(ctx).Info("Registering counterparty address for relayer.", "Address:", msg.Address, "Counterparty Address:", msg.CounterpartyAddress)

	return &types.MsgRegisterCounterpartyAddressResponse{}, nil
}

// EscrowPacketFee defines a rpc handler method for MsgEscrowPacketFee
// EscrowPacketFee is an open callback that may be called by any module/user that wishes to escrow funds in order to
// incentivize the relaying of the given packet.
func (k Keeper) EscrowPacketFee(goCtx context.Context, msg *types.MsgEscrowPacketFee) (*types.MsgEscrowPacketFeeResponse, error) {
	return &types.MsgEscrowPacketFeeResponse{}, nil
}
