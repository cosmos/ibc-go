package types

import (
	"fmt"

	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// IBC client events
const (
	AttributeKeyClientID        = "client_id"
	AttributeKeySubjectClientID = "subject_client_id"
	AttributeKeyClientType      = "client_type"

	// Deprecated: AttributeKeyConsensusHeight is deprecated and will be removed in a future release.
	// Please use AttributeKeyConsensusHeights instead.
	AttributeKeyConsensusHeight  = "consensus_height"
	AttributeKeyConsensusHeights = "consensus_heights"
	AttributeKeyHeader           = "header"
)

// IBC client events vars
var (
	EventTypeCreateClient         = "create_client"
	EventTypeUpdateClient         = "update_client"
	EventTypeUpgradeClient        = "upgrade_client"
	EventTypeSubmitMisbehaviour   = "client_misbehaviour"
	EventTypeUpdateClientProposal = "update_client_proposal"

	AttributeValueCategory = fmt.Sprintf("%s_%s", host.ModuleName, SubModuleName)
)
