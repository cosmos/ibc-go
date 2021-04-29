package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	abci "github.com/tendermint/tendermint/abci/types"
)

// StakingKeeper defines the contract expected by parent-chain ccv module.
// StakingKeeper is responsible for keeping track of latest validator set of all baby chains
type StakingKeeper interface {
	GetValidatorSetChanges(chainID string) []abci.ValidatorUpdate
	// This method is not required by CCV module explicitly but necessary for init protocol
	GetInitialValidatorSet(chainID string) []sdk.Tx
}

// RegistryKeeper defines the contract expected by parent-chain ccv module from a Registry Module that will keep track
// of chain creators and respective validator sets
// RegistryKeeper is responsible for verifying that chain creator is authorized to create a chain with given chain-id,
// as well as which validators are staking for a given chain.
type CNSKeeper interface {
	AuthorizeChainCreator(chainID, creator string)
	GetValidatorSet(chainID string) []sdk.ValAddress
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
	SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error
	ChanCloseInit(ctx sdk.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error
}

// PortKeeper defines the expected IBC port keeper
type PortKeeper interface {
	BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability
}

// TODO: Expected interfaces for distribution on parent and baby chains
