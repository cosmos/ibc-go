package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

<<<<<<< HEAD
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v5/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	solomachinetypes "github.com/cosmos/ibc-go/v5/modules/light-clients/06-solomachine/types"
	ibctmtypes "github.com/cosmos/ibc-go/v5/modules/light-clients/07-tendermint/types"
	localhosttypes "github.com/cosmos/ibc-go/v5/modules/light-clients/09-localhost/types"
=======
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	solomachinetypes "github.com/cosmos/ibc-go/v6/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
>>>>>>> c86d27f (chore: increment go mod to v6 (#2318))
)

// RegisterInterfaces registers x/ibc interfaces into protobuf Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	clienttypes.RegisterInterfaces(registry)
	connectiontypes.RegisterInterfaces(registry)
	channeltypes.RegisterInterfaces(registry)
	solomachinetypes.RegisterInterfaces(registry)
	ibctmtypes.RegisterInterfaces(registry)
	localhosttypes.RegisterInterfaces(registry)
	commitmenttypes.RegisterInterfaces(registry)
}
