package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// two methods for ack
// success - returns bool (if fwd relayer address empty, false)
// acknowledge will just marshl & return

// Success implements the Acknowledgement interface. The acknowledgement is
// considered successful if the forward relayer address is empty. Otherwise it is
// considered a failed acknowledgement.
func (ack IncentivizedAcknowledgement) Success() bool {
	return ack.ForwardRelayerAddress != ""
}

// Acknowledgement implements the Acknowledgement interface. It returns the
// acknowledgement serialised using JSON.
func (ack IncentivizedAcknowledgement) Acknowledgement() []byte {
	var SubModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return sdk.MustSortJSON(SubModuleCdc.MustMarshalJSON(&ack))
}
