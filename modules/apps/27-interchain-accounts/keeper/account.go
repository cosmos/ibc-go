package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

// The first step in registering an interchain account
// Binds a new port & calls OnChanOpenInit
func (k Keeper) InitInterchainAccount(ctx sdk.Context, connectionId, owner string) error {
	portId := k.GeneratePortId(owner, connectionId)

	// Check if the port is already bound
	isBound := k.IsBound(ctx, portId)
	if isBound == true {
		return sdkerrors.Wrap(types.ErrPortAlreadyBound, portId)
	}

	portCap := k.portKeeper.BindPort(ctx, portId)
	err := k.ClaimCapability(ctx, portCap, host.PortPath(portId))
	if err != nil {
		return err
	}

	counterParty := channeltypes.Counterparty{PortId: "ibcaccount", ChannelId: ""}
	order := channeltypes.Order(2)
	channelId, cap, err := k.channelKeeper.ChanOpenInit(ctx, order, []string{connectionId}, portId, portCap, counterParty, types.Version)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			channeltypes.EventTypeChannelOpenInit,
			sdk.NewAttribute(channeltypes.AttributeKeyPortID, portId),
			sdk.NewAttribute(channeltypes.AttributeKeyChannelID, channelId),
			sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, "ibcaccount"),
			sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, ""),
			sdk.NewAttribute(channeltypes.AttributeKeyConnectionID, connectionId),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
		),
	})

	_ = k.OnChanOpenInit(ctx, channeltypes.Order(2), []string{connectionId}, portId, channelId, cap, counterParty, types.Version)

	return err
}

// Register interchain account if it has not already been created
func (k Keeper) RegisterInterchainAccount(ctx sdk.Context, portId string) (types.IBCAccountI, error) {
	address := k.GenerateAddress(portId)
	account := k.accountKeeper.GetAccount(ctx, address)

	if account != nil {
		return nil, sdkerrors.Wrap(types.ErrAccountAlreadyExist, account.String())
	}

	interchainAccount := types.NewIBCAccount(
		authtypes.NewBaseAccountWithAddress(address),
		portId,
	)
	k.accountKeeper.NewAccount(ctx, interchainAccount)
	k.accountKeeper.SetAccount(ctx, interchainAccount)
	_ = k.SetInterchainAccountAddress(ctx, portId, interchainAccount.Address)

	return interchainAccount, nil
}

func (k Keeper) SetInterchainAccountAddress(ctx sdk.Context, portId string, address string) string {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyOwnerAccount(portId)
	store.Set(key, []byte(address))
	return address
}

func (k Keeper) GetInterchainAccountAddress(ctx sdk.Context, portId string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyOwnerAccount(portId)
	if !store.Has(key) {
		return "", sdkerrors.Wrap(types.ErrIBCAccountNotFound, portId)
	}

	interchainAccountAddr := string(store.Get(key))
	return interchainAccountAddr, nil
}

// Determine account's address that will be created.
func (k Keeper) GenerateAddress(identifier string) []byte {
	return tmhash.SumTruncated(append([]byte(identifier)))
}

func (k Keeper) GetIBCAccount(ctx sdk.Context, addr sdk.AccAddress) (types.IBCAccount, error) {
	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return types.IBCAccount{}, sdkerrors.Wrap(types.ErrIBCAccountNotFound, "there is no account")
	}

	ibcAcc, ok := acc.(*types.IBCAccount)
	if !ok {
		return types.IBCAccount{}, sdkerrors.Wrap(types.ErrIBCAccountNotFound, "account is not an IBC account")
	}
	return *ibcAcc, nil
}
