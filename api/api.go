package api

import (
	storetypes "cosmossdk.io/store/types"
	proto "github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Status represents the status of a client
type Status string

const (
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

type ClientStoreProvider interface {
	ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore
}

type LightClientModule interface {
	// RegisterStoreProvider is called by core IBC when a LightClientModule is added to the router.
	// It allows the LightClientModule to set a ClientStoreProvider which supplies isolated prefix client stores
	// to IBC light client instances.
	RegisterStoreProvider(storeProvider ClientStoreProvider)

	// Initialize is called upon client creation, it allows the client to perform validation on the initial consensus state and set the
	// client state, consensus state and any client-specific metadata necessary for correct light client operation in the provided client store.
	Initialize(ctx sdk.Context, clientID string, clientState, consensusState []byte) error

	// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
	// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
	// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
	// if the ClientMessage fails to verify.
	VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg ClientMessage) error

	// Checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
	// has already been verified.
	CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg ClientMessage) bool

	// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
	UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg ClientMessage)

	// UpdateState updates and stores as necessary any associated information for an IBC client, such as the ClientState and corresponding ConsensusState.
	// Upon successful update, a list of consensus heights is returned. It assumes the ClientMessage has already been verified.
	UpdateState(ctx sdk.Context, clientID string, clientMsg ClientMessage) []Height // TODO: change to concrete type

	// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
	// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
	VerifyMembership(
		ctx sdk.Context,
		clientID string,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		proof []byte,
		path MerklePath,
		value []byte,
	) error

	// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
	// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
	VerifyNonMembership(
		ctx sdk.Context,
		clientID string,
		height Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		proof []byte,
		path MerklePath,
	) error

	// Status must return the status of the client. Only Active clients are allowed to process packets.
	Status(ctx sdk.Context, clientID string) Status

	// TimestampAtHeight must return the timestamp for the consensus state associated with the provided height.
	TimestampAtHeight(
		ctx sdk.Context,
		clientID string,
		height Height,
	) (uint64, error)

	// CheckSubstituteAndUpdateState must verify that the provided substitute may be used to update the subject client.
	// The light client must set the updated client and consensus states within the clientStore for the subject client.
	// DEPRECATED: will be removed as performs internal functionality
	RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error

	// Upgrade functions
	// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last
	// height committed by the current revision. Clients are responsible for ensuring that the planned last
	// height of the current revision is somehow encoded in the proof verification process.
	// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty
	// may be cancelled or modified before the last planned height.
	// If the upgrade is verified, the upgraded client and consensus states must be set in the client store.
	// DEPRECATED: will be removed as performs internal functionality
	VerifyUpgradeAndUpdateState(
		ctx sdk.Context,
		clientID string,
		newClient []byte,
		newConsState []byte,
		upgradeClientProof,
		upgradeConsensusStateProof []byte,
	) error
}

// ClientMessage is an interface used to update an IBC client.
// The update may be done by a single header, a batch of headers, misbehaviour, or any type which when verified produces
// a change to state of the IBC client
type ClientMessage interface {
	proto.Message

	ClientType() string
	ValidateBasic() error
}
