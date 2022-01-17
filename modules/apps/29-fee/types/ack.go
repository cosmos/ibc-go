package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

// NewIncentivizedAcknowledgement creates a new instance of IncentivizedAcknowledgement
func NewIncentivizedAcknowledgement(relayer string, ack *codectypes.Any) IncentivizedAcknowledgement {
	return IncentivizedAcknowledgement{
		AppAcknowledgement:    ack,
		ForwardRelayerAddress: relayer,
	}
}

// Success implements the Acknowledgement interface. The acknowledgement is
// considered successful if the forward relayer address is empty. Otherwise it is
// considered a failed acknowledgement.
func (ack IncentivizedAcknowledgement) Success() bool {
	unpackedAck, err := channeltypes.UnpackAcknowledgement(ack.AppAcknowledgement)
	if err != nil {
		return false
	}

	return unpackedAck.Success()
}

// Acknowledgement implements the Acknowledgement interface. It returns the
// acknowledgement serialised using JSON.
func (ack IncentivizedAcknowledgement) Acknowledgement() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&ack))
}
