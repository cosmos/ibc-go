package types

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// instantiateMessage
type instantiateMessage struct {
	ClientState    *ClientState    `json:"client_state"`
	ConsensusState *ConsensusState `json:"consensus_state"`
}

// queryMsg is used to encode query messages
// omitempty tag is mandatory for JSON serialization
// to be compatible with Rust contract enum matching
type queryMsg struct {
	Status               *statusMsg               `json:"status,omitempty"`
	ExportMetadata       *exportMetadataMsg       `json:"export_metadata,omitempty"`
	TimestampAtHeight    *timestampAtHeightMsg    `json:"timestamp_at_height,omitempty"`
	VerifyClientMessage  *verifyClientMessageMsg  `json:"verify_client_message,omitempty"`
	VerifyMembership     *verifyMembershipMsg     `json:"verify_membership,omitempty"`
	VerifyNonMembership  *verifyNonMembershipMsg  `json:"verify_non_membership,omitempty"`
	CheckForMisbehaviour *checkForMisbehaviourMsg `json:"check_for_misbehaviour,omitempty"`
}

type (
	statusMsg            struct{}
	exportMetadataMsg    struct{}
	timestampAtHeightMsg struct {
		Height exported.Height `json:"height"`
	}
)

type verifyClientMessageMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}
type verifyMembershipMsg struct {
	Height           exported.Height `json:"height"`
	DelayTimePeriod  uint64          `json:"delay_time_period"`
	DelayBlockPeriod uint64          `json:"delay_block_period"`
	Proof            []byte          `json:"proof"`
	Path             exported.Path   `json:"path"`
	Value            []byte          `json:"value"`
}
type verifyNonMembershipMsg struct {
	Height           exported.Height `json:"height"`
	DelayTimePeriod  uint64          `json:"delay_time_period"`
	DelayBlockPeriod uint64          `json:"delay_block_period"`
	Proof            []byte          `json:"proof"`
	Path             exported.Path   `json:"path"`
}
type checkForMisbehaviourMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}

// sudoMsg is used to encode sudo messages
// omitempty tag is mandatory for JSON serialization
// to be compatible with Rust contract enum matching
type sudoMsg struct {
	UpdateState                   *updateStateMsg                   `json:"update_state,omitempty"`
	UpdateStateOnMisbehaviour     *updateStateOnMisbehaviourMsg     `json:"update_state_on_misbehaviour,omitempty"`
	VerifyUpgradeAndUpdateState   *verifyUpgradeAndUpdateStateMsg   `json:"verify_upgrade_and_update_state,omitempty"`
	CheckSubstituteAndUpdateState *checkSubstituteAndUpdateStateMsg `json:"check_substitute_and_update_state,omitempty"`
}

type updateStateMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}
type updateStateOnMisbehaviourMsg struct {
	ClientMessage *ClientMessage `json:"client_message"`
}
type verifyUpgradeAndUpdateStateMsg struct {
	UpgradeClientState         exported.ClientState    `json:"upgrade_client_state"`
	UpgradeConsensusState      exported.ConsensusState `json:"upgrade_consensus_state"`
	ProofUpgradeClient         []byte                  `json:"proof_upgrade_client"`
	ProofUpgradeConsensusState []byte                  `json:"proof_upgrade_consensus_state"`
}
type checkSubstituteAndUpdateStateMsg struct{}

// ContractResult defines the expected result returned by a contract call
type ContractResult interface {
	Validate() bool
	Error() string
}

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

type statusResult struct {
	contractResult
	Status exported.Status `json:"status"`
}

type exportMetadataResult struct {
	contractResult
	GenesisMetadata []clienttypes.GenesisMetadata `json:"genesis_metadata,omitempty"`
}

type timestampAtHeightResult struct {
	contractResult
	Timestamp uint64 `json:"timestamp"`
}

type checkForMisbehaviourResult struct {
	contractResult
	FoundMisbehaviour bool `json:"found_misbehaviour"`
}

type updateStateResult struct {
	contractResult
	Heights []clienttypes.Height `json:"heights"`
}
