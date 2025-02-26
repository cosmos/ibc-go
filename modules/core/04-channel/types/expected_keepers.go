package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
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
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		channel Channel,
	) error
	VerifyPacketCommitment(
		ctx sdk.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		commitmentBytes []byte,
	) error
	VerifyPacketAcknowledgement(
		ctx sdk.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		acknowledgement []byte,
	) error
	VerifyPacketReceiptAbsence(
		ctx sdk.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
	) error
	VerifyNextSequenceRecv(
		ctx sdk.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		nextSequenceRecv uint64,
	) error
}
