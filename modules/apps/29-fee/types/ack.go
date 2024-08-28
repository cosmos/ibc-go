package types

import (
	"encoding/json"
	"errors"
)

// NewFeeAcknowledgement creates a new instance of FeeAcknowledgement.
func NewFeeAcknowledgement(relayer string) FeeAcknowledgement {
	return FeeAcknowledgement{
		ForwardRelayerAddress: relayer,
	}
}

// Acknowledgement returns the fee acknowledgement encoded bytes.
// TODO: should accept and handle the v2 packet payload encoding schema.
func (ack FeeAcknowledgement) Acknowledgement() []byte {
	res, err := json.Marshal(&ack)
	if err != nil {
		panic(errors.New("cannot marshal fee acknowledgement into json"))
	}

	return res
}

// NewIncentivizedAcknowledgement creates a new instance of IncentivizedAcknowledgement
// Deprecated: use FeeAcknowledgement instead.
func NewIncentivizedAcknowledgement(relayer string, ack []byte, success bool) IncentivizedAcknowledgement {
	return IncentivizedAcknowledgement{
		AppAcknowledgement:    ack,
		ForwardRelayerAddress: relayer,
		UnderlyingAppSuccess:  success,
	}
}

// Success implements the Acknowledgement interface. The acknowledgement is
// considered successful if the forward relayer address is empty. Otherwise it is
// considered a failed acknowledgement.
// Deprecated: use FeeAcknowledgement instead.
func (ack IncentivizedAcknowledgement) Success() bool {
	return ack.UnderlyingAppSuccess
}

// Acknowledgement implements the Acknowledgement interface. It returns the
// acknowledgement serialised using JSON.
// Deprecated: use FeeAcknowledgement instead.
func (ack IncentivizedAcknowledgement) Acknowledgement() []byte {
	res, err := json.Marshal(&ack)
	if err != nil {
		panic(errors.New("cannot marshal acknowledgement into json"))
	}

	return res
}
