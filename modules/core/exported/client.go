package exported

import (
	"context"

	"github.com/cosmos/gogoproto/proto"
)

// Status represents the status of a client
type Status string

const (
	// Solomachine is used to indicate that the light client is a solo machine.
	Solomachine string = "06-solomachine"

	// Tendermint is used to indicate that the client uses the Tendermint Consensus Algorithm.
	Tendermint string = "07-tendermint"

	// Localhost is the client type for the localhost client.
	Localhost string = "09-localhost"

	// LocalhostClientID is the sentinel client ID for the localhost client.
	LocalhostClientID string = Localhost

	// Active is a status type of a client. An active client is allowed to be used.
	Active Status = "Active"

	// Frozen is a status type of a client. A frozen client is not allowed to be used.
	Frozen Status = "Frozen"

	// Expired is a status type of a client. An expired client is not allowed to be used.
	Expired Status = "Expired"

	// Unknown indicates there was an error in determining the status of a client.
	Unknown Status = "Unknown"

	// Unauthorized indicates that the client type is not registered as an allowed client type.
	Unauthorized Status = "Unauthorized"
)

// LightClientModule is an interface which core IBC uses to interact with light client modules.
// Light client modules must implement this interface to integrate with core IBC.
type LightClientModule interface {
	// Initialize is called upon client creation, it allows the client to perform validation on the client state and initial consensus state.
	// The light client module is responsible for setting any client-specific data in the store. This includes the client state,
	// initial consensus state and any associated metadata.
	Initialize(ctx context.Context, clientID string, clientState, consensusState []byte) error

	// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
	// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
	// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
	// if the ClientMessage fails to verify.
	VerifyClientMessage(ctx context.Context, clientID string, clientMsg ClientMessage) error

	// Checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
	// has already been verified.
	CheckForMisbehaviour(ctx context.Context, clientID string, clientMsg ClientMessage) bool

	// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
	UpdateStateOnMisbehaviour(ctx context.Context, clientID string, clientMsg ClientMessage)

	// UpdateState updates and stores as necessary any associated information for an IBC client, such as the ClientState and corresponding ConsensusState.
	// Upon successful update, a list of consensus heights is returned. It assumes the ClientMessage has already been verified.
	UpdateState(ctx context.Context, clientID string, clientMsg ClientMessage) []Height

	// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
	// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
	VerifyMembership(
		ctx context.Context,
		clientID string,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		proof []byte,
		path Path,
		value []byte,
	) error

	// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
	// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
	VerifyNonMembership(
		ctx context.Context,
		clientID string,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		proof []byte,
		path Path,
	) error

	// Status must return the status of the client. Only Active clients are allowed to process packets.
	Status(ctx context.Context, clientID string) Status

	// LatestHeight returns the latest height of the client. If no client is present for the provided client identifier a zero value height may be returned.
	LatestHeight(ctx context.Context, clientID string) Height

	// TimestampAtHeight must return the timestamp for the consensus state associated with the provided height.
	TimestampAtHeight(
		ctx context.Context,
		clientID string,
		height Height,
	) (uint64, error)

	// RecoverClient must verify that the provided substitute may be used to update the subject client.
	// The light client module must set the updated client and consensus states within the clientStore for the subject client.
	RecoverClient(ctx context.Context, clientID, substituteClientID string) error

	// Upgrade functions
	// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last
	// height committed by the current revision. Clients are responsible for ensuring that the planned last
	// height of the current revision is somehow encoded in the proof verification process.
	// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty
	// may be cancelled or modified before the last planned height.
	// If the upgrade is verified, the upgraded client and consensus states must be set in the client store.
	VerifyUpgradeAndUpdateState(
		ctx context.Context,
		clientID string,
		newClient []byte,
		newConsState []byte,
		upgradeClientProof,
		upgradeConsensusStateProof []byte,
	) error
}

// ClientState defines the required common functions for light clients.
type ClientState interface {
	proto.Message

	ClientType() string
	Validate() error
}

// ConsensusState is the state of the consensus process
type ConsensusState interface {
	proto.Message

	ClientType() string // Consensus kind

	// GetTimestamp returns the timestamp (in nanoseconds) of the consensus state
	//
	// Deprecated: GetTimestamp is not used outside of the light client implementations,
	// and therefore it doesn't need to be an interface function.
	GetTimestamp() uint64

	ValidateBasic() error
}

// ClientMessage is an interface used to update an IBC client.
// The update may be done by a single header, a batch of headers, misbehaviour, or any type which when verified produces
// a change to state of the IBC client
type ClientMessage interface {
	proto.Message

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
