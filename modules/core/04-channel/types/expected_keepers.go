package types

import (
	"context"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// ClientKeeper expected account IBC client keeper
type ClientKeeper interface {
	GetClientStatus(ctx context.Context, clientID string) exported.Status
	GetClientState(ctx context.Context, clientID string) (exported.ClientState, bool)
	GetClientConsensusState(ctx context.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
	GetClientLatestHeight(ctx context.Context, clientID string) clienttypes.Height
	GetClientTimestampAtHeight(ctx context.Context, clientID string, height exported.Height) (uint64, error)
}

// ConnectionKeeper expected account IBC connection keeper
type ConnectionKeeper interface {
	GetConnection(ctx context.Context, connectionID string) (connectiontypes.ConnectionEnd, bool)
	VerifyChannelState(
		ctx context.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		channel Channel,
	) error
	VerifyPacketCommitment(
		ctx context.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		commitmentBytes []byte,
	) error
	VerifyPacketAcknowledgement(
		ctx context.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		acknowledgement []byte,
	) error
	VerifyPacketReceiptAbsence(
		ctx context.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
	) error
	VerifyNextSequenceRecv(
		ctx context.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		nextSequenceRecv uint64,
	) error
	VerifyChannelUpgrade(
		ctx context.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		upgrade Upgrade,
	) error
	VerifyChannelUpgradeError(
		ctx context.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		errorReceipt ErrorReceipt,
	) error
}
