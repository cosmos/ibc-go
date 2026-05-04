package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterCodec registers the necessary x/tokenfactory interfaces and concrete types
// Required for legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateDenom{}, "tokenfactory/MsgCreateDenom", nil)
	cdc.RegisterConcrete(&MsgMint{}, "tokenfactory/MsgMint", nil)
	cdc.RegisterConcrete(&MsgBurn{}, "tokenfactory/MsgBurn", nil)
}

// RegisterInterfaces register the tokenfactory module interfaces to protobuf Any.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateDenom{},
		&MsgMint{},
		&MsgBurn{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
