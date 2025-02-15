package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// RegisterInterfaces registers the commitment interfaces to protobuf Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterInterface(
		"ibc.core.commitment.v1.Root",
		(*exported.Root)(nil),
	)
	registry.RegisterInterface(
		"ibc.core.commitment.v1.Prefix",
		(*exported.Prefix)(nil),
	)
	registry.RegisterInterface(
		"ibc.core.commitment.v1.Path",
		(*exported.Path)(nil),
	)

	registry.RegisterImplementations(
		(*exported.Root)(nil),
		&MerkleRoot{},
	)
	registry.RegisterImplementations(
		(*exported.Prefix)(nil),
		&MerklePrefix{},
	)
	registry.RegisterImplementations(
		(*exported.Path)(nil),
		&v2.MerklePath{},
	)
}
