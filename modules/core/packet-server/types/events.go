package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// IBC channel events
const (
	AttributeKeyChannelID = "channel_id"
)

// IBC channel events vars
var (
	EventTypeCreateChannel = "create_channel"

	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
