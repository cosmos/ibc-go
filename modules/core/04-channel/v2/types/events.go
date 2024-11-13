package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// IBC channel events
const (
	AttributeKeyChannelID             = "channel_id"
	AttributeKeyClientID              = "client_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
)

// IBC channel events vars
var (
	EventTypeCreateChannel        = "create_channel"
	EventTypeRegisterCounterparty = "register_counterparty"

	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
