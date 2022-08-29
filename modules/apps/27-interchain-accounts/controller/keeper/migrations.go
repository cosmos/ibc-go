package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
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
		owners, found := m.keeper.scopedKeeper.GetOwners(ctx, name)
		if !found {
			return sdkerrors.Wrapf(capabilitytypes.ErrCapabilityOwnersNotFound, "failed to find capability owners for: %s", name)
		}

		ibcOwner := capabilitytypes.NewOwner(host.ModuleName, name)
		if index, found := owners.Get(ibcOwner); !found && index != 0 {
			return sdkerrors.Wrapf(capabilitytypes.ErrCapabilityNotOwned, "expected capability owner: %s", host.ModuleName)
		}

		controllerOwner := capabilitytypes.NewOwner(types.SubModuleName, name)
		if index, found := owners.Get(controllerOwner); !found && index != 1 {
			return sdkerrors.Wrapf(capabilitytypes.ErrCapabilityNotOwned, "expected capability owner: %s", types.SubModuleName)
		}
	}

	return nil
}
