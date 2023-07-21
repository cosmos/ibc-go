package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ sdk.Msg = (*MsgStoreCode)(nil)

// MsgStoreCode creates a new MsgStoreCode instance
//
//nolint:interfacer
func NewMsgStoreCode(signer string, code []byte) *MsgStoreCode {
	return &MsgStoreCode{
		Signer: signer,
		Code:   code,
	}
}

// ValidateBasic implements sdk.Msg
func (m MsgStoreCode) ValidateBasic() error {
	if len(m.Code) == 0 {
		return ErrWasmEmptyCode
	}

	return nil
}

// GetSigners implements sdk.Msg
func (m MsgStoreCode) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

type QueryMsg struct {
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
	verifyClientMessageMsg struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
	verifyMembershipMsg struct {
		Height           exported.Height `json:"height"`
		DelayTimePeriod  uint64          `json:"delay_time_period"`
		DelayBlockPeriod uint64          `json:"delay_block_period"`
		Proof            []byte          `json:"proof"`
		Path             exported.Path   `json:"path"`
		Value            []byte          `json:"value"`
	}
	verifyNonMembershipMsg struct {
		Height           exported.Height `json:"height"`
		DelayTimePeriod  uint64          `json:"delay_time_period"`
		DelayBlockPeriod uint64          `json:"delay_block_period"`
		Proof            []byte          `json:"proof"`
		Path             exported.Path   `json:"path"`
	}
	checkForMisbehaviourMsg struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
)

type SudoMsg struct {
	Initialize                    *initializeMsg                    `json:"initialize,omitempty"`
	UpdateState                   *updateStateMsg                   `json:"update_state,omitempty"`
	UpdateStateOnMisbehaviour     *updateStateOnMisbehaviourMsg     `json:"update_state_on_misbehaviour,omitempty"`
	VerifyUpgradeAndUpdateState   *verifyUpgradeAndUpdateStateMsg   `json:"verify_upgrade_and_update_state,omitempty"`
	CheckSubstituteAndUpdateState *checkSubstituteAndUpdateStateMsg `json:"check_substitute_and_update_state,omitempty"`
}

type (
	initializeMsg struct {
		ClientState    *ClientState    `json:"client_state"`
		ConsensusState *ConsensusState `json:"consensus_state"`
	}
	updateStateMsg struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
	updateStateOnMisbehaviourMsg struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
	verifyUpgradeAndUpdateStateMsg struct {
		UpgradeClientState         exported.ClientState    `json:"upgrade_client_state"`
		UpgradeConsensusState      exported.ConsensusState `json:"upgrade_consensus_state"`
		ProofUpgradeClient         []byte                  `json:"proof_upgrade_client"`
		ProofUpgradeConsensusState []byte                  `json:"proof_upgrade_consensus_state"`
	}
	checkSubstituteAndUpdateStateMsg struct{}
)
