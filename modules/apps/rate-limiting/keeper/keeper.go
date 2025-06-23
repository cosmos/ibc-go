package keeper

import (
	"errors"
	"fmt"
	"strings"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
)

// Keeper maintains the link to storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.BinaryCodec

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	clientKeeper  types.ClientKeeper

	bankKeeper types.BankKeeper
	authority  string
}

// NewKeeper creates a new rate-limiting Keeper instance
func NewKeeper(cdc codec.BinaryCodec, storeService corestore.KVStoreService, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper, bankKeeper types.BankKeeper, authority string) *Keeper {
	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return &Keeper{
		cdc:          cdc,
		storeService: storeService,
		// Defaults to using the channel keeper as the ICS4Wrapper
		// This can be overridden later with WithICS4Wrapper (e.g. by the middleware stack wiring)
		ics4Wrapper:   channelKeeper,
		channelKeeper: channelKeeper,
		clientKeeper:  clientKeeper,
		bankKeeper:    bankKeeper,
		authority:     authority,
	}
}

// SetICS4Wrapper sets the ICS4Wrapper.
// It is used after the middleware is created since the keeper needs the underlying module's SendPacket capability,
// creating a dependency cycle.
func (k *Keeper) SetICS4Wrapper(ics4Wrapper porttypes.ICS4Wrapper) {
	k.ics4Wrapper = ics4Wrapper
}

// ICS4Wrapper returns the ICS4Wrapper to send packets downstream.
func (k *Keeper) ICS4Wrapper() porttypes.ICS4Wrapper {
	return k.ics4Wrapper
}

// GetAuthority returns the module's authority.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (*Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
