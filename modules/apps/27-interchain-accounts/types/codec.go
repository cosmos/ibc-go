package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// RegisterLegacyAminoCodec registers the account interfaces and concrete types on the
// provided LegacyAmino codec. These types are used for Amino JSON serialization
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterInterface((*InterchainAccountI)(nil), nil)
	cdc.RegisterConcrete(&InterchainAccount{}, "27-interchain-accounts/InterchainAccount", nil)
}

// RegisterInterface associates protoName with AccountI interface
// and creates a registry of it's concrete implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*authtypes.AccountI)(nil), &InterchainAccount{})
}

var (
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)
