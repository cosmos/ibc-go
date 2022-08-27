package types

import (
	fmt "fmt"

	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"
)

const (
	EventSendQuery = "sendQuery"
	
	AttributeQueryData           = "query_data"
	AttributeKeyTimeoutTimestamp = "query_timeout_timestamp"
	AttributeKeyQueryID          = "query_id"
	AttributeKeyTimeoutHeight    = "query_timeout_height"
	AttributeKeyQueryHeight      = "query_height"
)

var (
	AttributeValueCategory = fmt.Sprintf("%s_%s", host.ModuleName, ModuleName)
)