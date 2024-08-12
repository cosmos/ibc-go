package keeper

import (
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// ConnOpenInit initialises a connection attempt on chain A. The generated connection identifier
// is returned.
//
// NOTE: Msg validation verifies the supplied identifiers and ensures that the counterparty
// connection identifier is empty.
func (k Keeper) ConnOpenInit(
	ctx sdk.Context,
	clientID string,
	counterparty types.Counterparty, // counterpartyPrefix, counterpartyClientIdentifier
	version *types.Version,
	delayPeriod uint64,
) (string, error) {
	versions := types.GetCompatibleVersions()
	if version != nil {
		if !types.IsSupportedVersion(types.GetCompatibleVersions(), version) {
			return "", sdkerrors.Wrap(types.ErrInvalidVersion, "version is not supported")
		}

		versions = []exported.Version{version}
	}

	clientState, found := k.clientKeeper.GetClientState(ctx, clientID)
	if !found {
		return "", sdkerrors.Wrapf(clienttypes.ErrClientNotFound, "clientID (%s)", clientID)
	}

	if status := k.clientKeeper.GetClientStatus(ctx, clientState, clientID); status != exported.Active {
		return "", sdkerrors.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	connectionID := k.GenerateConnectionIdentifier(ctx)
	if err := k.addConnectionToClient(ctx, clientID, connectionID); err != nil {
		return "", err
	}

	// connection defines chain A's ConnectionEnd
	connection := types.NewConnectionEnd(types.INIT, clientID, counterparty, types.ExportedVersionsToProto(versions), delayPeriod)
	k.SetConnection(ctx, connectionID, connection)

	k.Logger(ctx).Info("connection state updated", "connection-id", connectionID, "previous-state", "NONE", "new-state", "INIT")

	defer func() {
		telemetry.IncrCounter(1, "ibc", "connection", "open-init")
	}()

	EmitConnectionOpenInitEvent(ctx, connectionID, clientID, counterparty)

	return connectionID, nil
}

// ConnOpenTry relays notice of a connection attempt on chain A to chain B (this
// code is executed on chain B).
//
// NOTE:
//   - Here chain A acts as the counterparty
//   - Identifiers are checked on msg validation
func (k Keeper) ConnOpenTry(
	ctx sdk.Context,
	counterparty types.Counterparty, // counterpartyConnectionIdentifier, counterpartyPrefix and counterpartyClientIdentifier
	delayPeriod uint64,
	clientID string, // clientID of chainA
	_ exported.ClientState, // clientState that chainA has for chainB
	counterpartyVersions []exported.Version, // supported versions of chain A
	proofInit []byte, // proof that chainA stored connectionEnd in state (on ConnOpenInit)
	_ []byte, // proof that chainA stored a light client of chainB
	_ []byte, // proof that chainA stored chainB's consensus state at consensus height
	proofHeight exported.Height, // height at which relayer constructs proof of A storing connectionEnd in state
	_ exported.Height, // latest height of chain B which chain A has stored in its chain B client
) (string, error) {
	// generate a new connection
	connectionID := k.GenerateConnectionIdentifier(ctx)

	// expectedConnection defines Chain A's ConnectionEnd
	// NOTE: chain A's counterparty is chain B (i.e where this code is executed)
	// NOTE: chainA and chainB must have the same delay period
	prefix := k.GetCommitmentPrefix()
	expectedCounterparty := types.NewCounterparty(clientID, "", commitmenttypes.NewMerklePrefix(prefix.Bytes()))
	expectedConnection := types.NewConnectionEnd(types.INIT, counterparty.ClientId, expectedCounterparty, types.ExportedVersionsToProto(counterpartyVersions), delayPeriod)

	// chain B picks a version from Chain A's available versions that is compatible
	// with Chain B's supported IBC versions. PickVersion will select the intersection
	// of the supported versions and the counterparty versions.
	version, err := types.PickVersion(types.GetCompatibleVersions(), counterpartyVersions)
	if err != nil {
		return "", err
	}

	// connection defines chain B's ConnectionEnd
	connection := types.NewConnectionEnd(types.TRYOPEN, clientID, counterparty, []*types.Version{version}, delayPeriod)

	// Check that ChainA committed expectedConnectionEnd to its state
	if err := k.VerifyConnectionState(
		ctx, connection, proofHeight, proofInit, counterparty.ConnectionId,
		expectedConnection,
	); err != nil {
		return "", err
	}

	// store connection in chainB state
	if err := k.addConnectionToClient(ctx, clientID, connectionID); err != nil {
		return "", sdkerrors.Wrapf(err, "failed to add connection with ID %s to client with ID %s", connectionID, clientID)
	}

	k.SetConnection(ctx, connectionID, connection)
	k.Logger(ctx).Info("connection state updated", "connection-id", connectionID, "previous-state", "NONE", "new-state", "TRYOPEN")

	defer func() {
		telemetry.IncrCounter(1, "ibc", "connection", "open-try")
	}()

	EmitConnectionOpenTryEvent(ctx, connectionID, clientID, counterparty)

	return connectionID, nil
}

// ConnOpenAck relays acceptance of a connection open attempt from chain B back
// to chain A (this code is executed on chain A).
//
// NOTE: Identifiers are checked on msg validation.
func (k Keeper) ConnOpenAck(
	ctx sdk.Context,
	connectionID string,
	_ exported.ClientState, // client state for chainA on chainB
	version *types.Version, // version that ChainB chose in ConnOpenTry
	counterpartyConnectionID string,
	proofTry []byte, // proof that connectionEnd was added to ChainB state in ConnOpenTry
	_ []byte, // proof of client state on chainB for chainA
	_ []byte, // proof that chainB has stored ConsensusState of chainA on its client
	proofHeight exported.Height, // height that relayer constructed proofTry
	_ exported.Height, // latest height of chainA that chainB has stored on its chainA client
) error {
	// Retrieve connection
	connection, found := k.GetConnection(ctx, connectionID)
	if !found {
		return sdkerrors.Wrap(types.ErrConnectionNotFound, connectionID)
	}

	// verify the previously set connection state
	if connection.State != types.INIT {
		return sdkerrors.Wrapf(
			types.ErrInvalidConnectionState,
			"connection state is not INIT (got %s)", connection.State.String(),
		)
	}

	// ensure selected version is supported
	if !types.IsSupportedVersion(types.ProtoVersionsToExported(connection.Versions), version) {
		return sdkerrors.Wrapf(
			types.ErrInvalidConnectionState,
			"the counterparty selected version %s is not supported by versions selected on INIT", version,
		)
	}

	prefix := k.GetCommitmentPrefix()
	expectedCounterparty := types.NewCounterparty(connection.ClientId, connectionID, commitmenttypes.NewMerklePrefix(prefix.Bytes()))
	expectedConnection := types.NewConnectionEnd(types.TRYOPEN, connection.Counterparty.ClientId, expectedCounterparty, []*types.Version{version}, connection.DelayPeriod)

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyConnectionState(
		ctx, connection, proofHeight, proofTry, counterpartyConnectionID,
		expectedConnection,
	); err != nil {
		return err
	}

	k.Logger(ctx).Info("connection state updated", "connection-id", connectionID, "previous-state", "INIT", "new-state", "OPEN")

	defer func() {
		telemetry.IncrCounter(1, "ibc", "connection", "open-ack")
	}()

	// Update connection state to Open
	connection.State = types.OPEN
	connection.Versions = []*types.Version{version}
	connection.Counterparty.ConnectionId = counterpartyConnectionID
	k.SetConnection(ctx, connectionID, connection)

	EmitConnectionOpenAckEvent(ctx, connectionID, connection)

	return nil
}

// ConnOpenConfirm confirms opening of a connection on chain A to chain B, after
// which the connection is open on both chains (this code is executed on chain B).
//
// NOTE: Identifiers are checked on msg validation.
func (k Keeper) ConnOpenConfirm(
	ctx sdk.Context,
	connectionID string,
	proofAck []byte, // proof that connection opened on ChainA during ConnOpenAck
	proofHeight exported.Height, // height that relayer constructed proofAck
) error {
	// Retrieve connection
	connection, found := k.GetConnection(ctx, connectionID)
	if !found {
		return sdkerrors.Wrap(types.ErrConnectionNotFound, connectionID)
	}

	// Check that connection state on ChainB is on state: TRYOPEN
	if connection.State != types.TRYOPEN {
		return sdkerrors.Wrapf(
			types.ErrInvalidConnectionState,
			"connection state is not TRYOPEN (got %s)", connection.State.String(),
		)
	}

	prefix := k.GetCommitmentPrefix()
	expectedCounterparty := types.NewCounterparty(connection.ClientId, connectionID, commitmenttypes.NewMerklePrefix(prefix.Bytes()))
	expectedConnection := types.NewConnectionEnd(types.OPEN, connection.Counterparty.ClientId, expectedCounterparty, connection.Versions, connection.DelayPeriod)

	// Check that connection on ChainA is open
	if err := k.VerifyConnectionState(
		ctx, connection, proofHeight, proofAck, connection.Counterparty.ConnectionId,
		expectedConnection,
	); err != nil {
		return err
	}

	// Update ChainB's connection to Open
	connection.State = types.OPEN
	k.SetConnection(ctx, connectionID, connection)
	k.Logger(ctx).Info("connection state updated", "connection-id", connectionID, "previous-state", "TRYOPEN", "new-state", "OPEN")

	defer func() {
		telemetry.IncrCounter(1, "ibc", "connection", "open-confirm")
	}()

	EmitConnectionOpenConfirmEvent(ctx, connectionID, connection)

	return nil
}
