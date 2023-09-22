package types

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// instantiateMessage is the message that is sent to the contract's instantiate entry point.
type instantiateMessage struct {
	ClientState    *ClientState    `json:"client_state"`
	ConsensusState *ConsensusState `json:"consensus_state"`
}

// queryMsg is used to encode messages that are sent to the contract's query entry point.
// The json omitempty tag is mandatory since it omits any empty (default initialized) fields from the encoded JSON,
// this is required in order to be compatible with Rust's enum matching as used in the contract.
type queryMsg struct {
	Status               *statusMsg               `json:"status,omitempty"`
	ExportMetadata       *exportMetadataMsg       `json:"export_metadata,omitempty"`
	TimestampAtHeight    *timestampAtHeightMsg    `json:"timestamp_at_height,omitempty"`
	VerifyClientMessage  *verifyClientMessageMsg  `json:"verify_client_message,omitempty"`
	VerifyMembership     *verifyMembershipMsg     `json:"verify_membership,omitempty"`
	VerifyNonMembership  *verifyNonMembershipMsg  `json:"verify_non_membership,omitempty"`
	CheckForMisbehaviour *checkForMisbehaviourMsg `json:"check_for_misbehaviour,omitempty"`
}

// statusMsg is a queryMsg sent to the contract to query the status of the wasm client.
type statusMsg struct{}

// exportMetadataMsg is a queryMsg sent to the contract to query the exported metadata of the wasm client.
type exportMetadataMsg struct{}

// timestampAtHeightMsg is a queryMsg sent to the contract to query the timestamp at a given height.
type timestampAtHeightMsg struct {
	Height exported.Height `json:"height"`
}

// verifyClientMessageMsg is a queryMsg sent to the contract to verify a client message.
type verifyClientMessageMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}

// verifyMembershipMsg is a queryMsg sent to the contract to verify a membership proof.
type verifyMembershipMsg struct {
	Height           exported.Height `json:"height"`
	DelayTimePeriod  uint64          `json:"delay_time_period"`
	DelayBlockPeriod uint64          `json:"delay_block_period"`
	Proof            []byte          `json:"proof"`
	Path             exported.Path   `json:"path"`
	Value            []byte          `json:"value"`
}

// verifyNonMembershipMsg is a queryMsg sent to the contract to verify a non-membership proof.
type verifyNonMembershipMsg struct {
	Height           exported.Height `json:"height"`
	DelayTimePeriod  uint64          `json:"delay_time_period"`
	DelayBlockPeriod uint64          `json:"delay_block_period"`
	Proof            []byte          `json:"proof"`
	Path             exported.Path   `json:"path"`
}

// checkForMisbehaviourMsg is a queryMsg sent to the contract to check for misbehaviour.
type checkForMisbehaviourMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}

// sudoMsg is used to encode messages that are sent to the contract's sudo entry point.
// The json omitempty tag is mandatory since it omits any empty (default initialized) fields from the encoded JSON,
// this is required in order to be compatible with Rust's enum matching as used in the contract.
type sudoMsg struct {
	UpdateState                   *updateStateMsg                   `json:"update_state,omitempty"`
	UpdateStateOnMisbehaviour     *updateStateOnMisbehaviourMsg     `json:"update_state_on_misbehaviour,omitempty"`
	VerifyUpgradeAndUpdateState   *verifyUpgradeAndUpdateStateMsg   `json:"verify_upgrade_and_update_state,omitempty"`
	CheckSubstituteAndUpdateState *checkSubstituteAndUpdateStateMsg `json:"check_substitute_and_update_state,omitempty"`
}

// updateStateMsg is a sudoMsg sent to the contract to update the client state.
type updateStateMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}

// updateStateOnMisbehaviourMsg is a sudoMsg sent to the contract to update its state on misbehaviour.
type updateStateOnMisbehaviourMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}

// verifyUpgradeAndUpdateStateMsg is a sudoMsg sent to the contract to verify an upgrade and update its state.
type verifyUpgradeAndUpdateStateMsg struct {
	UpgradeClientState         exported.ClientState    `json:"upgrade_client_state"`
	UpgradeConsensusState      exported.ConsensusState `json:"upgrade_consensus_state"`
	ProofUpgradeClient         []byte                  `json:"proof_upgrade_client"`
	ProofUpgradeConsensusState []byte                  `json:"proof_upgrade_consensus_state"`
}

// checkSubstituteAndUpdateStateMsg is a sudoMsg sent to the contract to check a given substitute client and update to its state.
type checkSubstituteAndUpdateStateMsg struct{}

// ContractResult defines the expected interface a Result returned by a contract call is expected to implement.
type ContractResult interface {
	Validate() bool
	Error() string
}

// contractResult is the default implementation of the ContractResult interface and the default return type of any contract call
// that does not require a custom return type.
type contractResult struct {
	IsValid  bool   `json:"is_valid,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

func (r contractResult) Validate() bool {
	return r.IsValid
}

func (r contractResult) Error() string {
	return r.ErrorMsg
}

// statusResult is the expected return type of the statusMsg query. It returns the status of the wasm client.
type statusResult struct {
	contractResult
	Status exported.Status `json:"status"`
}

// exportMetadataResult is the expected return type of the exportMetadataMsg query. It returns the exported metadata of the wasm client.
type exportMetadataResult struct {
	contractResult
	GenesisMetadata []clienttypes.GenesisMetadata `json:"genesis_metadata,omitempty"`
}

// timestampAtHeightResult is the expected return type of the timestampAtHeightMsg query. It returns the timestamp for a light client
// at a given height.
type timestampAtHeightResult struct {
	contractResult
	Timestamp uint64 `json:"timestamp"`
}

// checkForMisbehaviourResult is the expected return type of the checkForMisbehaviourMsg query. It returns a boolean indicating
// if misbehaviour was detected.
type checkForMisbehaviourResult struct {
	contractResult
	FoundMisbehaviour bool `json:"found_misbehaviour"`
}

// updateStateResult is the expected return type of the updateStateMsg sudo call. It returns the updated consensus heights.
type updateStateResult struct {
	contractResult
	Heights []clienttypes.Height `json:"heights"`
}
