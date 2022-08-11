package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v5/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	solomachinetypes "github.com/cosmos/ibc-go/v5/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v5/modules/light-clients/07-tendermint"
)

// RegisterInterfaces registers x/ibc interfaces into protobuf Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	clienttypes.RegisterInterfaces(registry)
	connectiontypes.RegisterInterfaces(registry)
	channeltypes.RegisterInterfaces(registry)
	solomachinetypes.RegisterInterfaces(registry)
	ibctm.RegisterInterfaces(registry)
	commitmenttypes.RegisterInterfaces(registry)
}
