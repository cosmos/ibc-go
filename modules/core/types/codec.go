package types

import (
	coreregistry "cosmossdk.io/core/registry"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
)

// RegisterInterfaces registers ibc types against interfaces using the global InterfaceRegistry.
// Note: The localhost client is created by ibc core and thus requires explicit type registration.
func RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	clienttypes.RegisterInterfaces(registry)
	connectiontypes.RegisterInterfaces(registry)
	channeltypes.RegisterInterfaces(registry)
	commitmenttypes.RegisterInterfaces(registry)
}
