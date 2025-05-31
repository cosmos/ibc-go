package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary rate-limiting interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddRateLimit{}, "ratelimit/MsgAddRateLimit", nil)
	cdc.RegisterConcrete(&MsgUpdateRateLimit{}, "ratelimit/MsgUpdateRateLimit", nil)
	cdc.RegisterConcrete(&MsgRemoveRateLimit{}, "ratelimit/MsgRemoveRateLimit", nil)
	cdc.RegisterConcrete(&MsgResetRateLimit{}, "ratelimit/MsgResetRateLimit", nil)
}

// RegisterInterfaces registers the rate-limiting interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgAddRateLimit{},
		&MsgUpdateRateLimit{},
		&MsgRemoveRateLimit{},
		&MsgResetRateLimit{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// ModuleCdc references the global rate-limiting module codec. Note, the codec should
// ONLY be used in certain instances of tests and for JSON encoding.
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
