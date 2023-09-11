package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// IBC client events
const (
	AttributeKeyClientID          = "client_id"
	AttributeKeySubjectClientID   = "subject_client_id"
	AttributeKeyClientType        = "client_type"
	AttributeKeyConsensusHeight   = "consensus_height"
	AttributeKeyConsensusHeights  = "consensus_heights"
	AttributeKeyHeader            = "header"
	AttributeKeyUpgradeStore      = "upgrade_store"
	AttributeKeyUpgradePlanHeight = "upgrade_plan_height"
	AttributeKeyUpgradePlanTitle  = "title"
)

// IBC client events vars
var (
	EventTypeCreateClient               = "create_client"
	EventTypeUpdateClient               = "update_client"
	EventTypeUpgradeClient              = "upgrade_client"
	EventTypeSubmitMisbehaviour         = "client_misbehaviour"
	EventTypeUpgradeChain               = "upgrade_chain"
	EventTypeScheduleIBCSoftwareUpgrade = "schedule_ibc_software_upgrade"
	EventTypeRecoverClient              = "recover_client"

	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
