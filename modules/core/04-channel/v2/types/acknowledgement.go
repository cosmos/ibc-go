package types

import (
	errorsmod "cosmossdk.io/errors"
)

// NewAcknowledgement creates a new Acknowledgement instance
func NewAcknowledgement(recvSuccess bool, appAcknowledgements [][]byte) Acknowledgement {
	return Acknowledgement{
		RecvSuccess:         recvSuccess,
		AppAcknowledgements: appAcknowledgements,
	}
}

// ValidateBasic validates the acknowledgment
func (a Acknowledgement) ValidateBasic() error {
	if len(a.GetAppAcknowledgements()) == 0 {
		return errorsmod.Wrap(ErrInvalidAcknowledgement, "acknowledgement cannot be empty")
	}
	for _, ack := range a.GetAppAcknowledgements() {
		if len(ack) == 0 {
			return errorsmod.Wrap(ErrInvalidAcknowledgement, "app acknowledgement cannot be empty")
		}
	}
	return nil
}
