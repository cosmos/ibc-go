package types

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v3/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// ClientType is beefy.
func (cs ClientState) ClientType() string {
	return exported.Beefy
}

// GetLatestHeight returns latest block height.
func (cs ClientState) GetLatestHeight() exported.Height {
	return clienttypes.NewHeight(0, cs.LatestBeefyHeight)
}


// Validate performs basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if cs.LatestBeefyHeight == 0 {
		return ErrInvalidHeaderHeight
	}

	return nil
}

// Initialize will check that initial consensus state is equal to the latest consensus state of the initial client.
func (cs ClientState) Initialize(_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore, consState exported.ConsensusState) error {
	if _, ok := consState.(*ConsensusState); !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T",
			&ConsensusState{}, consState)
	}
	return nil
}

// Status returns the status of the beefy client.
// The client may be:
// - Active: if frozen sequence is 0
// - Frozen: otherwise beefy client is frozen
func (cs ClientState) Status(_ sdk.Context, _ sdk.KVStore, _ codec.BinaryCodec) exported.Status {
	if cs.FrozenHeight > 0 {
		return exported.Frozen
	}

	return exported.Active
}

// ExportMetadata is a no-op since beefy does not store any metadata in client store
func (cs ClientState) ExportMetadata(_ sdk.KVStore) []exported.GenesisMetadata {
	return nil
}

// ZeroCustomFields returns a ClientState that is a copy of the current ClientState
// with all client customizable fields zeroed out
func (cs ClientState) ZeroCustomFields() exported.ClientState {
	// copy over all chain-specified fields
	// and leave custom fields empty
	return &ClientState{
		LatestBeefyHeight: 0,
	}
}


// VerifyClientState verifies a proof of the client state of the running chain
// stored on the target machine
func (cs ClientState) VerifyClientState(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	prefix exported.Prefix,
	counterpartyClientIdentifier string,
	proof []byte,
	clientState exported.ClientState,
) error {
	if clientState == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "client state cannot be empty")
	}

	_, ok := clientState.(*ClientState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "invalid client type %T, expected %T", clientState, &ClientState{})
	}

	clientPrefixedPath := commitmenttypes.NewMerklePath(host.FullClientStatePath(counterpartyClientIdentifier))
	path, err := commitmenttypes.ApplyPrefix(prefix, clientPrefixedPath)
	if err != nil {
		return err
	}

	csEncoded, err := Encode(clientState)
	if err != nil {
		return sdkerrors.Wrap(err, "clientState could not be scale encoded")
	}

	beefyProof, provingConsensusState, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	// TODO: verify use of keyPath as key in trie verification
	key, err := Encode(path.GetKeyPath())
	if err != nil {
		return sdkerrors.Wrap(err, "keyPath could not be scale encoded")
	}

	isVerified, err := trie.VerifyProof(beefyProof, provingConsensusState.Root, []trie.Pair{{Key: key, Value: csEncoded}})
	if err != nil {
		return fmt.Errorf("error verifying proof: %v", err.Error())
	}

	if !isVerified {
		return sdkerrors.Wrap(err, "unable to verify client state")
	}

	return nil
}

// VerifyClientConsensusState verifies a proof of the consensus state of the
// Tendermint client stored on the target machine.
func (cs ClientState) VerifyClientConsensusState(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	counterpartyClientIdentifier string,
	consensusHeight exported.Height,
	prefix exported.Prefix,
	proof []byte,
	consensusState exported.ConsensusState,
) error {
	if consensusState == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "consensus state cannot be empty")
	}

	_, ok := consensusState.(*ConsensusState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "invalid client type %T, expected %T", consensusState, &ConsensusState{})
	}

	csEncoded, err := Encode(consensusState)
	if err != nil {
		return sdkerrors.Wrap(err, "consensusState could not be scale encoded")
	}

	clientPrefixedPath := commitmenttypes.NewMerklePath(host.FullConsensusStatePath(counterpartyClientIdentifier, consensusHeight))
	path, err := commitmenttypes.ApplyPrefix(prefix, clientPrefixedPath)
	if err != nil {
		return err
	}

	beefyProof, provingConsensusState, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	// TODO: verify use of keyPath as key in trie verification
	key, err := Encode(path.GetKeyPath())
	if err != nil {
		return sdkerrors.Wrap(err, "keyPath could not be scale encoded")
	}

	isVerified, err := trie.VerifyProof(beefyProof, provingConsensusState.Root, []trie.Pair{{Key: key, Value: csEncoded}})
	if err != nil {
		return fmt.Errorf("error verifying proof: %v", err.Error())
	}

	if !isVerified {
		return sdkerrors.Wrap(err, "unable to verify client consensus state")
	}

	return nil
}

// VerifyPacketCommitment verifies a proof of an outgoing packet commitment at
// the specified port, specified channel, and specified sequence.
func (cs ClientState) VerifyPacketCommitment(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	_ uint64,
	_ uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
) error {
	beefyProof, consensusState, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	commitmentPath := commitmenttypes.NewMerklePath(host.PacketCommitmentPath(portID, channelID, sequence))
	path, err := commitmenttypes.ApplyPrefix(prefix, commitmentPath)
	if err != nil {
		return err
	}

	// TODO: verify use of keyPath as key in trie verification
	key, err := Encode(path.GetKeyPath())
	if err != nil {
		return sdkerrors.Wrap(err, "keyPath could not be scale encoded")
	}

	isVerified, err := trie.VerifyProof(beefyProof, consensusState.Root, []trie.Pair{{Key: key, Value: commitmentBytes}})
	if err != nil {
		return fmt.Errorf("error verifying proof: %v", err.Error())
	}

	if !isVerified {
		return sdkerrors.Wrap(err, "unable to verify client consensus state")
	}
	return nil
}

type BeefyProof [][]byte

// produceVerificationArgs performs the basic checks on the arguments that are
// shared between the verification functions and returns the unmarshalled
// merkle proof, the consensus state and an error if one occurred.
func produceVerificationArgs(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	cs ClientState,
	height exported.Height,
	prefix exported.Prefix,
	proof []byte,
) (beefyProof BeefyProof, consensusState *ConsensusState, err error) {
	if cs.GetLatestHeight().LT(height) {
		return BeefyProof{}, nil, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	if proof == nil {
		return BeefyProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "proof cannot be empty")
	}

	if prefix == nil {
		return BeefyProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidPrefix, "prefix cannot be empty")
	}

	_, ok := prefix.(*commitmenttypes.MerklePrefix)
	if !ok {
		return BeefyProof{}, nil, sdkerrors.Wrapf(commitmenttypes.ErrInvalidPrefix, "invalid prefix type %T, expected *MerklePrefix", prefix)
	}

	err = Decode(proof, &beefyProof)
	if err != nil {
		return BeefyProof{}, nil, sdkerrors.Wrap(err, "proof couldn't be decoded into BeefyProof struct")
	}

	consensusState, err = GetConsensusState(store, cdc, height)
	if err != nil {
		return BeefyProof{}, nil, sdkerrors.Wrap(err, "please ensure the proof was constructed against a height that exists on the client")
	}

	return beefyProof, consensusState, nil
}

// VerifyConnectionState verifies a proof of the connection state of the
// specified connection end stored on the target machine.
func (cs ClientState) VerifyConnectionState(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	prefix exported.Prefix,
	proof []byte,
	connectionID string,
	connectionEnd exported.ConnectionI,
) error {
	beefyProof, consensusState, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	connectionPath := commitmenttypes.NewMerklePath(host.ConnectionPath(connectionID))
	path, err := commitmenttypes.ApplyPrefix(prefix, connectionPath)
	if err != nil {
		return err
	}

	connection, ok := connectionEnd.(connectiontypes.ConnectionEnd)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid connection type %T", connectionEnd)
	}

	// TODO: verify use of keyPath as key in trie verification
	key, err := Encode(path.GetKeyPath())
	if err != nil {
		return sdkerrors.Wrap(err, "keyPath could not be scale encoded")
	}

	connEncoded, err := Encode(connection)
	if err != nil {
		return sdkerrors.Wrap(err, "connection state could not be scale encoded")
	}

	isVerified, err := trie.VerifyProof(beefyProof, consensusState.Root, []trie.Pair{{Key: key, Value: connEncoded}})
	if err != nil {
		return fmt.Errorf("error verifying proof: %v", err.Error())
	}

	if !isVerified {
		return sdkerrors.Wrap(err, "unable to verify client consensus state")
	}
	return nil
}

// VerifyPacketAcknowledgement verifies a proof of an incoming packet
// acknowledgement at the specified port, specified channel, and specified sequence.
func (cs ClientState) VerifyPacketAcknowledgement(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	_ uint64,
	_ uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
) error {
	beefyProof, consensusState, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	ackPath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementPath(portID, channelID, sequence))
	path, err := commitmenttypes.ApplyPrefix(prefix, ackPath)
	if err != nil {
		return err
	}

	// TODO: verify use of keyPath as key in trie verification
	key, err := Encode(path.GetKeyPath())
	if err != nil {
		return sdkerrors.Wrap(err, "keyPath could not be scale encoded")
	}

	isVerified, err := trie.VerifyProof(beefyProof, consensusState.Root, []trie.Pair{{Key: key, Value: channeltypes.CommitAcknowledgement(acknowledgement)}})
	if err != nil {
		return fmt.Errorf("error verifying proof: %v", err.Error())
	}

	if !isVerified {
		return sdkerrors.Wrap(err, "unable to verify client consensus state")
	}

	return nil
}

func (cs ClientState) VerifyChannelState(
	store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, proof []byte, portID, channelID string, channel exported.ChannelI) error {
	beefyProof, consensusState, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	channelPath := commitmenttypes.NewMerklePath(host.ChannelPath(portID, channelID))
	path, err := commitmenttypes.ApplyPrefix(prefix, channelPath)
	if err != nil {
		return err
	}

	channelEnd, ok := channel.(channeltypes.Channel)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid channel type %T", channel)
	}

	chanEncoded, err := Encode(channelEnd)
	if err != nil {
		return sdkerrors.Wrap(err, "channel end could not be scale encoded")
	}

	// TODO: verify use of keyPath as key in trie verification
	key, err := Encode(path.GetKeyPath())
	if err != nil {
		return sdkerrors.Wrap(err, "keyPath could not be scale encoded")
	}

	isVerified, err := trie.VerifyProof(beefyProof, consensusState.Root, []trie.Pair{{Key: key, Value: chanEncoded}})
	if err != nil {
		return fmt.Errorf("error verifying proof: %v", err.Error())
	}

	if !isVerified {
		return sdkerrors.Wrap(err, "unable to verify client consensus state")
	}

	return nil
}

func (cs ClientState) VerifyPacketReceiptAbsence(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	_ uint64,
	_ uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	) error {
	_, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	receiptPath := commitmenttypes.NewMerklePath(host.PacketReceiptPath(portID, channelID, sequence))
	_, err = commitmenttypes.ApplyPrefix(prefix, receiptPath)
	if err != nil {
		return err
	}

	// TODO: use parity/trie proofOfNonExistence for receipt absense
	return nil
}

func (cs ClientState) VerifyNextSequenceRecv(
	_ sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	_ uint64,
	_ uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	nextSequenceRecv uint64,
	) error {
	beefyProof, consensusState, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	nextSequenceRecvPath := commitmenttypes.NewMerklePath(host.NextSequenceRecvPath(portID, channelID))
	path, err := commitmenttypes.ApplyPrefix(prefix, nextSequenceRecvPath)
	if err != nil {
		return err
	}

	key, err := Encode(path)
	if err != nil {
		return sdkerrors.Wrap(err, "next sequence recv path could not be scale encoded")
	}

	bz := sdk.Uint64ToBigEndian(nextSequenceRecv)

	isVerified, err := trie.VerifyProof(beefyProof, consensusState.Root, []trie.Pair{{Key: key, Value: bz}})
	if err != nil {
		return fmt.Errorf("error verifying proof: %v", err.Error())
	}

	if !isVerified {
		return sdkerrors.Wrap(err, "unable to verify client consensus state")
	}

	return nil
}

func (cs ClientState) VerifyUpgradeAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, newClient exported.ClientState, newConsState exported.ConsensusState, proofUpgradeClient, proofUpgradeConsState []byte) (exported.ClientState, exported.ConsensusState, error) {
	panic("implement me")
}

func (cs ClientState) CheckHeaderAndUpdateState(context sdk.Context, codec codec.BinaryCodec, store sdk.KVStore, header exported.Header) (exported.ClientState, exported.ConsensusState, error) {
	panic("implement me")
}

func (cs ClientState) CheckMisbehaviourAndUpdateState(context sdk.Context, codec codec.BinaryCodec, store sdk.KVStore, misbehaviour exported.Misbehaviour) (exported.ClientState, error) {
	panic("implement me")
}

func (cs ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore, substituteClientStore sdk.KVStore, substituteClient exported.ClientState) (exported.ClientState, error) {
	panic("implement me")
}