package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channelv2types "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

var _ codectypes.UnpackInterfacesMessage = (*GenesisState)(nil)

// DefaultGenesisState returns the ibc module's default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		ClientGenesis:     clienttypes.DefaultGenesisState(),
		ClientV2Genesis:   clientv2types.DefaultGenesisState(),
		ConnectionGenesis: connectiontypes.DefaultGenesisState(),
		ChannelGenesis:    channeltypes.DefaultGenesisState(),
		ChannelV2Genesis:  channelv2types.DefaultGenesisState(),
	}
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (gs GenesisState) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return gs.ClientGenesis.UnpackInterfaces(unpacker)
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs *GenesisState) Validate() error {
	if err := gs.ClientGenesis.Validate(); err != nil {
		return err
	}

	if err := gs.ClientV2Genesis.Validate(); err != nil {
		return err
	}

	if err := gs.ConnectionGenesis.Validate(); err != nil {
		return err
	}

	if err := gs.ChannelGenesis.Validate(); err != nil {
		return err
	}

	return gs.ChannelV2Genesis.Validate()
}
