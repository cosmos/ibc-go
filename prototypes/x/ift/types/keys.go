package types

import "cosmossdk.io/collections"

const (
	ModuleName = "ift"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// Store key prefixes
var (
	ParamsKey          = collections.NewPrefix(0)
	IFTBridgePrefix    = collections.NewPrefix(1)
	PendingTransferKey = collections.NewPrefix(2)
)
