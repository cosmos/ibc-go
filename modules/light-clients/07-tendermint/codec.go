package tendermint

import (
	coreregistry "cosmossdk.io/core/registry"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// RegisterInterfaces registers the tendermint concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	registry.RegisterImplementations(
		(*exported.ClientState)(nil),
		&ClientState{},
	)
	registry.RegisterImplementations(
		(*exported.ConsensusState)(nil),
		&ConsensusState{},
	)
	registry.RegisterImplementations(
		(*exported.ClientMessage)(nil),
		&Header{},
	)
	registry.RegisterImplementations(
		(*exported.ClientMessage)(nil),
		&Misbehaviour{},
	)
}
