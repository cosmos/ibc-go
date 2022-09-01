package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// AssertChannelCapabilityMigrations checks that all channel capabilities generated using the interchain accounts controller port prefix
// are owned by the controller submodule and ibc.
func (m Migrator) AssertChannelCapabilityMigrations(ctx sdk.Context) error {
	for _, channel := range m.keeper.GetAllActiveChannels(ctx) {
		name := host.ChannelCapabilityPath(channel.PortId, channel.ChannelId)
		cap, found := m.keeper.scopedKeeper.GetCapability(ctx, name)
		if !found {
			return sdkerrors.Wrapf(capabilitytypes.ErrCapabilityNotFound, "failed to find capability: %s", name)
		}

		isAuthenticated := m.keeper.scopedKeeper.AuthenticateCapability(ctx, cap, name)
		if !isAuthenticated {
			return sdkerrors.Wrapf(capabilitytypes.ErrCapabilityNotOwned, "expected capability owner: %s", types.SubModuleName)
		}
	}

	return nil
}
