package keeper

import (
	"fmt"
	"strings"

	baseapp "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
)

// Keeper defines the IBC transfer keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	hook types.IBCAccountHooks

	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	accountKeeper types.AccountKeeper

	scopedKeeper capabilitykeeper.ScopedKeeper

	msgRouter *baseapp.MsgServiceRouter
	memKey    sdk.StoreKey
}

// NewKeeper creates a new interchain account Keeper instance
func NewKeeper(
	memKey sdk.StoreKey,
	cdc codec.BinaryCodec, key sdk.StoreKey,
	channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	accountKeeper types.AccountKeeper, scopedKeeper capabilitykeeper.ScopedKeeper, msgRouter *baseapp.MsgServiceRouter, hook types.IBCAccountHooks,
) Keeper {
	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		accountKeeper: accountKeeper,
		scopedKeeper:  scopedKeeper,
		msgRouter:     msgRouter,
		memKey:        memKey,
		hook:          hook,
	}
}

func (k Keeper) SerializeCosmosTx(cdc codec.BinaryCodec, data interface{}) ([]byte, error) {
	msgs := make([]sdk.Msg, 0)
	switch data := data.(type) {
	case sdk.Msg:
		msgs = append(msgs, data)
	case []sdk.Msg:
		msgs = append(msgs, data...)
	default:
		return nil, types.ErrInvalidOutgoingData
	}

	msgAnys := make([]*codectypes.Any, len(msgs))

	for i, msg := range msgs {
		var err error
		msgAnys[i], err = codectypes.NewAnyWithValue(msg)
		if err != nil {
			return nil, err
		}
	}

	txBody := &types.IBCTxBody{
		Messages: msgAnys,
	}

	txRaw := &types.IBCTxRaw{
		BodyBytes: cdc.MustMarshal(txBody),
	}

	bz, err := cdc.Marshal(txRaw)
	if err != nil {
		return nil, err
	}

	return bz, nil
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s-%s", host.ModuleName, types.ModuleName))
}

// IsBound checks if the interchain account module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the port Keeper's BindPort function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	// Set the portID into our store so we can retrieve it later
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.PortKey), []byte(portID))

	cap := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}

// GetPort returns the portID for the interchain accounts module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get([]byte(types.PortKey)))
}

// ClaimCapability allows the transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// Utility function for parsing the connection number from the connection-id
func getConnectionNumber(connectionId string) string {
	ss := strings.Split(connectionId, "-")
	return ss[len(ss)-1]
}

func (k Keeper) GeneratePortId(owner, connectionId string) string {
	ownerId := strings.TrimSpace(owner)
	connectionNumber := getConnectionNumber(connectionId)
	portId := types.IcaPrefix + connectionNumber + "-" + ownerId
	return portId
}

func (k Keeper) SetActiveChannel(ctx sdk.Context, portId, channelId string) error {
	store := ctx.KVStore(k.storeKey)

	key := types.KeyActiveChannel(portId)
	store.Set(key, []byte(channelId))
	return nil
}

func (k Keeper) GetActiveChannel(ctx sdk.Context, portId string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyActiveChannel(portId)
	if !store.Has(key) {
		return "", false
	}

	activeChannel := string(store.Get(key))
	return activeChannel, true
}

// IsActiveChannel returns true if there exists an active channel for
// the provided portID and false otherwise.
func (k Keeper) IsActiveChannel(ctx sdk.Context, portId string) bool {
	_, found := k.GetActiveChannel(ctx, portId)
	return found
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, cap, name)
}
