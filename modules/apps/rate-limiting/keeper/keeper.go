package keeper

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	porttypes "github.com/cosmos/ibc-go/v11/modules/core/05-port/types"
)

// Keeper maintains the link to storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.BinaryCodec
	addressCodec address.Codec
	Schema       collections.Schema

	// PendingSendPackets stores packets whose send flow was applied and may need
	// to be reverted on timeout or error acknowledgement. The key order is
	// (channelID, denom, sequence) to support efficient channel+denom range resets.
	PendingSendPackets collections.KeySet[collections.Triple[string, string, uint64]]
	// PendingReceivePackets stores packets whose receive flow was applied and may
	// need to be reverted when an async acknowledgement fails. The key order is
	// (channelID, denom, sequence) to support efficient channel+denom range resets.
	PendingReceivePackets collections.KeySet[collections.Triple[string, string, uint64]]

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	clientKeeper  types.ClientKeeper

	bankKeeper types.BankKeeper
	authority  string
}

// NewKeeper creates a new rate-limiting Keeper instance
func NewKeeper(cdc codec.BinaryCodec, addressCodec address.Codec, storeService corestore.KVStoreService, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper, bankKeeper types.BankKeeper, authority string) *Keeper {
	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	sb := collections.NewSchemaBuilder(storeService)
	pendingPacketKeyCodec := collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.Uint64Key)
	k := Keeper{
		cdc:          cdc,
		addressCodec: addressCodec,
		storeService: storeService,

		PendingSendPackets:    collections.NewKeySet(sb, types.PendingSendPacketsKey, "pending_send_packets", pendingPacketKeyCodec),
		PendingReceivePackets: collections.NewKeySet(sb, types.PendingReceivePacketsKey, "pending_receive_packets", pendingPacketKeyCodec),

		// Defaults to using the channel keeper as the ICS4Wrapper
		// This can be overridden later with WithICS4Wrapper (e.g. by the middleware stack wiring)
		ics4Wrapper:   channelKeeper,
		channelKeeper: channelKeeper,
		clientKeeper:  clientKeeper,
		bankKeeper:    bankKeeper,
		authority:     authority,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return &k
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
