package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// ClientKeeper expected account IBC client keeper
type ClientKeeper interface {
	GetClientStatus(ctx sdk.Context, clientID string) exported.Status
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
	GetClientLatestHeight(ctx sdk.Context, clientID string) clienttypes.Height
	GetClientTimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error)
}

// ConnectionKeeper expected account IBC connection keeper
type ConnectionKeeper interface {
	GetConnection(ctx sdk.Context, connectionID string) (connectiontypes.ConnectionEnd, bool)
	VerifyChannelState(
		ctx sdk.Context,
		portID,
		channelID string,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		channel Channel,
	) error
	VerifyPacketCommitment(
		ctx sdk.Context,
		portID,
		channelID string,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		sequence uint64,
		commitmentBytes []byte,
	) error
	VerifyPacketAcknowledgement(
		ctx sdk.Context,
		portID,
		channelID string,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		sequence uint64,
		acknowledgement []byte,
	) error
	VerifyPacketReceiptAbsence(
		ctx sdk.Context,
		portID,
		channelID string,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		sequence uint64,
	) error
	VerifyNextSequenceRecv(
		ctx sdk.Context,
		portID,
		channelID string,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		nextSequenceRecv uint64,
	) error
	VerifyChannelUpgrade(
		ctx sdk.Context,
		portID,
		channelID string,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		upgrade Upgrade,
	) error
	VerifyChannelUpgradeError(
		ctx sdk.Context,
		portID,
		channelID string,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		errorReceipt ErrorReceipt,
	) error
}

// PortKeeper expected account IBC port keeper
type PortKeeper interface {
	Authenticate(ctx sdk.Context, key *capabilitytypes.Capability, portID string) bool
}
