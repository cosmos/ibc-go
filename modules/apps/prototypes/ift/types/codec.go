package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/ift interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgRegisterIFTBridge{}, "ibc/applications/ift/v1/MsgRegisterIFTBridge")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveIFTBridge{}, "ibc/applications/ift/v1/MsgRemoveIFTBridge")
	legacy.RegisterAminoMsg(cdc, &MsgIFTTransfer{}, "ibc/applications/ift/v1/MsgIFTTransfer")
	legacy.RegisterAminoMsg(cdc, &MsgIFTMint{}, "ibc/applications/ift/v1/MsgIFTMint")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "ibc/applications/ift/v1/MsgUpdateParams")
}

// RegisterInterfaces registers the x/ift interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterIFTBridge{},
		&MsgRemoveIFTBridge{},
		&MsgIFTTransfer{},
		&MsgIFTMint{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
