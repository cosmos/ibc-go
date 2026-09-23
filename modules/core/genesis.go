// SPDX-License-Identifier: Apache-2.0

package ibc

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	client "github.com/cosmos/ibc-go/v11/modules/core/02-client"
	clientv2 "github.com/cosmos/ibc-go/v11/modules/core/02-client/v2"
	connection "github.com/cosmos/ibc-go/v11/modules/core/03-connection"
	channel "github.com/cosmos/ibc-go/v11/modules/core/04-channel"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	channelv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2"
	"github.com/cosmos/ibc-go/v11/modules/core/keeper"
	"github.com/cosmos/ibc-go/v11/modules/core/types"
)

// InitGenesis initializes the ibc state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, gs *types.GenesisState) {
	client.InitGenesis(ctx, k.ClientKeeper, gs.ClientGenesis)
	clientv2.InitGenesis(ctx, k.ClientV2Keeper, gs.ClientV2Genesis)
	connection.InitGenesis(ctx, k.ConnectionKeeper, gs.ConnectionGenesis)
	channel.InitGenesis(ctx, k.ChannelKeeper, gs.ChannelGenesis)
	channelv2.InitGenesis(ctx, k.ChannelKeeperV2, gs.ChannelV2Genesis)
	restoreChannelAliases(ctx, k)
}

// restoreChannelAliases sets the IBC v2 counterparty and the alias of every OPEN UNORDERED
// channel, like the channel handshake and the v11 migration do, so that v2 packets can keep
// using the v1 channel identifiers. Neither is part of the genesis state.
func restoreChannelAliases(ctx sdk.Context, k keeper.Keeper) {
	k.ChannelKeeper.IterateChannels(ctx, func(ic channeltypes.IdentifiedChannel) bool {
		counterparty, ok := k.ChannelKeeper.GetV2Counterparty(ctx, ic.PortId, ic.ChannelId)
		if !ok {
			return false
		}
		connection, ok := k.ConnectionKeeper.GetConnection(ctx, ic.ConnectionHops[0])
		if !ok {
			panic(fmt.Errorf("connection %s not found for channel %s", ic.ConnectionHops[0], ic.ChannelId))
		}

		k.ClientV2Keeper.SetClientCounterparty(ctx, ic.ChannelId, counterparty)
		k.ChannelKeeperV2.SetClientForAlias(ctx, ic.ChannelId, connection.ClientId)
		return false
	})
}

// ExportGenesis returns the ibc exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	channelGenesis := channel.ExportGenesis(ctx, k.ChannelKeeper)
	channelV2Genesis := channelv2.ExportGenesis(ctx, k.ChannelKeeperV2)

	// v2 packets sent on a v1 channel identifier (channel aliasing) are stored under the
	// channel ID, which the client based channel v2 export doesn't cover. Their send
	// sequence is shared with the v1 channel and already exported with it.
	for _, ch := range channelGenesis.Channels {
		if _, ok := k.ChannelKeeperV2.GetClientForAlias(ctx, ch.ChannelId); ok {
			channelv2.ExportPacketState(ctx, k.ChannelKeeperV2, &channelV2Genesis, ch.ChannelId)
		}
	}

	return &types.GenesisState{
		ClientGenesis:     client.ExportGenesis(ctx, k.ClientKeeper),
		ClientV2Genesis:   clientv2.ExportGenesis(ctx, k.ClientV2Keeper),
		ConnectionGenesis: connection.ExportGenesis(ctx, k.ConnectionKeeper),
		ChannelGenesis:    channelGenesis,
		ChannelV2Genesis:  channelV2Genesis,
	}
}
