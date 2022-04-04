package exported

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/gogo/protobuf/proto"
)

// Status represents the status of a client
type Status string

const (
	// TypeClientMisbehaviour is the shared evidence misbehaviour type
	TypeClientMisbehaviour string = "client_misbehaviour"

	// Solomachine is used to indicate that the light client is a solo machine.
	Solomachine string = "06-solomachine"

	// Tendermint is used to indicate that the client uses the Tendermint Consensus Algorithm.
	Tendermint string = "07-tendermint"

	// Active is a status type of a client. An active client is allowed to be used.
	Active Status = "Active"

	// Frozen is a status type of a client. A frozen client is not allowed to be used.
	Frozen Status = "Frozen"

	// Expired is a status type of a client. An expired client is not allowed to be used.
	Expired Status = "Expired"

	// Unknown indicates there was an error in determining the status of a client.
	Unknown Status = "Unknown"
)

// ClientState defines the required common functions for light clients.
type ClientState interface {
	proto.Message

	ClientType() string
	GetLatestHeight() Height
	Validate() error

	// Initialization function
	// Clients must validate the initial consensus state, and may store any client-specific metadata
	// necessary for correct light client operation
	Initialize(sdk.Context, codec.BinaryCodec, sdk.KVStore, ConsensusState) error

	// Status function
	// Clients must return their status. Only Active clients are allowed to process packets.
	Status(ctx sdk.Context, clientStore sdk.KVStore, cdc codec.BinaryCodec) Status

	// Checks for evidence of a misbehaviour in Header or Misbehaviour type
	CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, msg ClientMessage) bool

	// Genesis function
	ExportMetadata(sdk.KVStore) []GenesisMetadata

	// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
	UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg ClientMessage)

	// VerifyClientMessage verifies a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
	VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg ClientMessage) error

	// UpdateState updates and stores as necessary any associated information for an IBC client, such as the ClientState and corresponding ConsensusState.
	// An error is returned if ClientMessage is of type Misbehaviour
	UpdateState(sdk.Context, codec.BinaryCodec, sdk.KVStore, ClientMessage) error

	// Update and Misbehaviour functions
	CheckMisbehaviourAndUpdateState(sdk.Context, codec.BinaryCodec, sdk.KVStore, ClientMessage) (ClientState, error)
	CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore, substituteClientStore sdk.KVStore, substituteClient ClientState) (ClientState, error)

	// Upgrade functions
	// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last
	// height committed by the current revision. Clients are responsible for ensuring that the planned last
	// height of the current revision is somehow encoded in the proof verification process.
	// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty
	// may be cancelled or modified before the last planned height.
	// If the upgrade is verified, the upgraded client and consensus states must be set in the client store.
	VerifyUpgradeAndUpdateState(
		ctx sdk.Context,
		cdc codec.BinaryCodec,
		store sdk.KVStore,
		newClient ClientState,
		newConsState ConsensusState,
		proofUpgradeClient,
		proofUpgradeConsState []byte,
	) error
	// Utility function that zeroes out any client customizable fields in client state
	// Ledger enforced fields are maintained while all custom fields are zero values
	// Used to verify upgrades
	ZeroCustomFields() ClientState

	// State verification functions

	VerifyClientState(
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		prefix Prefix,
		counterpartyClientIdentifier string,
		proof []byte,
		clientState ClientState,
	) error
	VerifyClientConsensusState(
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		counterpartyClientIdentifier string,
		consensusHeight Height,
		prefix Prefix,
		proof []byte,
		consensusState ConsensusState,
	) error
	VerifyConnectionState(
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		prefix Prefix,
		proof []byte,
		connectionID string,
		connectionEnd ConnectionI,
	) error
	VerifyChannelState(
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		prefix Prefix,
		proof []byte,
		portID,
		channelID string,
		channel ChannelI,
	) error
	VerifyPacketCommitment(
		ctx sdk.Context,
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		prefix Prefix,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		commitmentBytes []byte,
	) error
	VerifyPacketAcknowledgement(
		ctx sdk.Context,
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		prefix Prefix,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		acknowledgement []byte,
	) error
	VerifyPacketReceiptAbsence(
		ctx sdk.Context,
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		prefix Prefix,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
	) error
	VerifyNextSequenceRecv(
		ctx sdk.Context,
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		prefix Prefix,
		proof []byte,
		portID,
		channelID string,
		nextSequenceRecv uint64,
	) error
}

// ConsensusState is the state of the consensus process
type ConsensusState interface {
	proto.Message

	ClientType() string // Consensus kind

	// GetTimestamp returns the timestamp (in nanoseconds) of the consensus state
	GetTimestamp() uint64

	ValidateBasic() error
}

// ClientMessage is an interface used to update an IBC client.
// The update may be done by a single header, a batch of headers, misbehaviour, or any type which when verified produces
// a change to state of the IBC client
type ClientMessage interface {
	proto.Message

	GetHeight() Height
	ClientType() string
	ValidateBasic() error
}

// Height is a wrapper interface over clienttypes.Height
// all clients must use the concrete implementation in types
type Height interface {
	IsZero() bool
	LT(Height) bool
	LTE(Height) bool
	EQ(Height) bool
	GT(Height) bool
	GTE(Height) bool
	GetRevisionNumber() uint64
	GetRevisionHeight() uint64
	Increment() Height
	Decrement() (Height, bool)
	String() string
}

// GenesisMetadata is a wrapper interface over clienttypes.GenesisMetadata
// all clients must use the concrete implementation in types
type GenesisMetadata interface {
	// return store key that contains metadata without clientID-prefix
	GetKey() []byte
	// returns metadata value
	GetValue() []byte
}

// String returns the string representation of a client status.
func (s Status) String() string {
	return string(s)
}
