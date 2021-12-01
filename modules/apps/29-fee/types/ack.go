package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewMsgRegisterCounterpartyAddress creates a new instance of MsgRegisterCounterpartyAddress
func NewIncentivizedAcknowledgement(relayer string, ack []byte) IncentivizedAcknowledgement {
	return IncentivizedAcknowledgement{
		Result:                ack,
		ForwardRelayerAddress: relayer,
	}
}

// Acknowledgement implements the Acknowledgement interface. It returns the
// acknowledgement serialised using JSON.
func (ack IncentivizedAcknowledgement) Acknowledgement() []byte {
	var SubModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return sdk.MustSortJSON(SubModuleCdc.MustMarshalJSON(&ack))
}
