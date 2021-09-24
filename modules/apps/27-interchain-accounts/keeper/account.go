package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// InitInterchainAccount is the entry point to registering an interchain account.
// It generates a new port identifier using the owner address, connection identifier,
// and counterparty connection identifier. It will bind to the port identifier and
// call 04-channel 'ChanOpenInit'. An error is returned if the port identifier is
// already in use. Gaining access to interchain accounts whose channels have closed
// cannot be done with this function. A regular MsgChanOpenInit must be used.
func (k Keeper) InitInterchainAccount(ctx sdk.Context, connectionID, counterpartyConnectionID, owner string) error {
	portID, err := types.GeneratePortID(owner, connectionID, counterpartyConnectionID)
	if err != nil {
		return err
	}

	if k.IsBound(ctx, portID) {
		return sdkerrors.Wrap(types.ErrPortAlreadyBound, portID)
	}

	cap := k.BindPort(ctx, portID)
	if err := k.ClaimCapability(ctx, cap, host.PortPath(portID)); err != nil {
		return sdkerrors.Wrap(err, "unable to bind to newly generated portID")
	}

	msg := channeltypes.NewMsgChannelOpenInit(portID, types.VersionPrefix, channeltypes.ORDERED, []string{connectionID}, types.PortID, types.ModuleName)
	handler := k.msgRouter.Handler(msg)
	if _, err := handler(ctx, msg); err != nil {
		return err
	}

	return nil
}

// RegisterInterchainAccount attempts to create a new account using the provided address and stores it in state keyed by the provided port identifier
// If an account for the provided address already exists this function returns early (no-op)
func (k Keeper) RegisterInterchainAccount(ctx sdk.Context, accAddr sdk.AccAddress, portID string) {
	if acc := k.accountKeeper.GetAccount(ctx, accAddr); acc != nil {
		return
	}

	interchainAccount := types.NewInterchainAccount(
		authtypes.NewBaseAccountWithAddress(accAddr),
		portID,
	)

	k.accountKeeper.NewAccount(ctx, interchainAccount)
	k.accountKeeper.SetAccount(ctx, interchainAccount)
	k.SetInterchainAccountAddress(ctx, portID, interchainAccount.Address)
}

func (k Keeper) GetInterchainAccount(ctx sdk.Context, addr sdk.AccAddress) (types.InterchainAccount, error) {
	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return types.InterchainAccount{}, sdkerrors.Wrap(types.ErrInterchainAccountNotFound, "there is no account")
	}

	interchainAccount, ok := acc.(*types.InterchainAccount)
	if !ok {
		return types.InterchainAccount{}, sdkerrors.Wrap(types.ErrInterchainAccountNotFound, "account is not an interchain account")
	}
	return *interchainAccount, nil
}
