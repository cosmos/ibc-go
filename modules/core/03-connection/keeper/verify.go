package keeper

import (
	"math"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// VerifyClientState verifies a proof of a client state of the running machine
// stored on the target machine
func (k *Keeper) VerifyClientState(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	clientState exported.ClientState,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	merklePath := commitmenttypes.NewMerklePath(host.FullClientStateKey(connection.Counterparty.ClientId))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	bz, err := k.cdc.MarshalInterface(clientState)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height,
		0, 0, // skip delay period checks for non-packet processing verification
		proof, merklePath, bz,
	); err != nil {
		return errorsmod.Wrapf(err, "failed client state verification for target client: %s", clientID)
	}

	return nil
}

// VerifyClientConsensusState verifies a proof of the consensus state of the
// specified client stored on the target machine.
func (k *Keeper) VerifyClientConsensusState(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	consensusHeight exported.Height,
	proof []byte,
	consensusState exported.ConsensusState,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	merklePath := commitmenttypes.NewMerklePath(host.FullConsensusStateKey(connection.Counterparty.ClientId, consensusHeight))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	bz, err := k.cdc.MarshalInterface(consensusState)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height,
		0, 0, // skip delay period checks for non-packet processing verification
		proof, merklePath, bz,
	); err != nil {
		return errorsmod.Wrapf(err, "failed consensus state verification for client (%s)", clientID)
	}

	return nil
}

// VerifyConnectionState verifies a proof of the connection state of the
// specified connection end stored on the target machine.
func (k *Keeper) VerifyConnectionState(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	connectionID string,
	counterpartyConnection types.ConnectionEnd, // opposite connection
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	merklePath := commitmenttypes.NewMerklePath(host.ConnectionKey(connectionID))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(&counterpartyConnection)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height,
		0, 0, // skip delay period checks for non-packet processing verification
		proof, merklePath, bz,
	); err != nil {
		return errorsmod.Wrapf(err, "failed connection state verification for client (%s)", clientID)
	}

	return nil
}

// VerifyChannelState verifies a proof of the channel state of the specified
// channel end, under the specified port, stored on the target machine.
func (k *Keeper) VerifyChannelState(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	channel channeltypes.Channel,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(portID, channelID))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(&channel)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height,
		0, 0, // skip delay period checks for non-packet processing verification
		proof, merklePath, bz,
	); err != nil {
		return errorsmod.Wrapf(err, "failed channel state verification for client (%s)", clientID)
	}

	return nil
}

// VerifyPacketCommitment verifies a proof of an outgoing packet commitment at
// the specified port, specified channel, and specified sequence.
func (k *Keeper) VerifyPacketCommitment(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	// get time and block delays
	timeDelay := connection.DelayPeriod
	blockDelay := k.getBlockDelay(ctx, connection)

	merklePath := commitmenttypes.NewMerklePath(host.PacketCommitmentKey(portID, channelID, sequence))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height, timeDelay, blockDelay, proof, merklePath, commitmentBytes,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet commitment verification for client (%s)", clientID)
	}

	return nil
}

// VerifyPacketAcknowledgement verifies a proof of an incoming packet
// acknowledgement at the specified port, specified channel, and specified sequence.
func (k *Keeper) VerifyPacketAcknowledgement(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	// get time and block delays
	timeDelay := connection.DelayPeriod
	blockDelay := k.getBlockDelay(ctx, connection)

	merklePath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementKey(portID, channelID, sequence))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height, timeDelay, blockDelay,
		proof, merklePath, channeltypes.CommitAcknowledgement(acknowledgement),
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet acknowledgement verification for client (%s)", clientID)
	}

	return nil
}

// VerifyPacketReceiptAbsence verifies a proof of the absence of an
// incoming packet receipt at the specified port, specified channel, and
// specified sequence.
func (k *Keeper) VerifyPacketReceiptAbsence(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	// get time and block delays
	timeDelay := connection.DelayPeriod
	blockDelay := k.getBlockDelay(ctx, connection)

	merklePath := commitmenttypes.NewMerklePath(host.PacketReceiptKey(portID, channelID, sequence))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyNonMembership(
		ctx, clientID, height, timeDelay, blockDelay, proof, merklePath,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet receipt absence verification for client (%s)", clientID)
	}

	return nil
}

// VerifyNextSequenceRecv verifies a proof of the next sequence number to be
// received of the specified channel at the specified port.
func (k *Keeper) VerifyNextSequenceRecv(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	nextSequenceRecv uint64,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	// get time and block delays
	timeDelay := connection.DelayPeriod
	blockDelay := k.getBlockDelay(ctx, connection)

	merklePath := commitmenttypes.NewMerklePath(host.NextSequenceRecvKey(portID, channelID))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height,
		timeDelay, blockDelay,
		proof, merklePath, sdk.Uint64ToBigEndian(nextSequenceRecv),
	); err != nil {
		return errorsmod.Wrapf(err, "failed next sequence receive verification for client (%s)", clientID)
	}

	return nil
}

// VerifyChannelUpgradeError verifies a proof of the provided upgrade error receipt.
func (k *Keeper) VerifyChannelUpgradeError(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	errorReceipt channeltypes.ErrorReceipt,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	merklePath := commitmenttypes.NewMerklePath(host.ChannelUpgradeErrorKey(portID, channelID))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(&errorReceipt)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, height,
		0, 0, // skip delay period checks for non-packet processing verification
		proof, merklePath, bz,
	); err != nil {
		return errorsmod.Wrapf(err, "failed upgrade error receipt verification for client (%s)", clientID)
	}

	return nil
}

// VerifyChannelUpgrade verifies the proof that a particular proposed upgrade has been stored in the upgrade path.
func (k *Keeper) VerifyChannelUpgrade(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	proofHeight exported.Height,
	upgradeProof []byte,
	portID,
	channelID string,
	upgrade channeltypes.Upgrade,
) error {
	clientID := connection.ClientId
	if status := k.clientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	merklePath := commitmenttypes.NewMerklePath(host.ChannelUpgradeKey(portID, channelID))
	merklePath, err := commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	if err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(&upgrade)
	if err != nil {
		return err
	}

	if err := k.clientKeeper.VerifyMembershipProof(
		ctx, clientID, proofHeight,
		0, 0, // skip delay period checks for non-packet processing verification
		upgradeProof, merklePath, bz,
	); err != nil {
		return errorsmod.Wrapf(err, "failed upgrade verification for client (%s) on channel (%s)", clientID, channelID)
	}

	return nil
}

// getBlockDelay calculates the block delay period from the time delay of the connection
// and the maximum expected time per block.
func (k *Keeper) getBlockDelay(ctx sdk.Context, connection types.ConnectionEnd) uint64 {
	// expectedTimePerBlock should never be zero, however if it is then return a 0 block delay for safety
	// as the expectedTimePerBlock parameter was not set.
	expectedTimePerBlock := k.GetParams(ctx).MaxExpectedTimePerBlock
	if expectedTimePerBlock == 0 {
		return 0
	}
	// calculate minimum block delay by dividing time delay period
	// by the expected time per block. Round up the block delay.
	timeDelay := connection.DelayPeriod
	return uint64(math.Ceil(float64(timeDelay) / float64(expectedTimePerBlock)))
}
