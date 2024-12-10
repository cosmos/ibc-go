package types

import (
	errorsmod "cosmossdk.io/errors"
)

// NewAcknowledgement creates a new Acknowledgement containing the provided app acknowledgements.
func NewAcknowledgement(appAcknowledgements ...[]byte) Acknowledgement {
	return Acknowledgement{AppAcknowledgements: appAcknowledgements}
}

// Validate performs a basic validation of the acknowledgement
func (ack Acknowledgement) Validate() error {
	if len(ack.AppAcknowledgements) != 1 {
		return errorsmod.Wrap(ErrInvalidAcknowledgement, "app acknowledgements must be of length one")
	}

	for _, ack := range ack.AppAcknowledgements {
		if len(ack) == 0 {
			return errorsmod.Wrap(ErrInvalidAcknowledgement, "app acknowledgement cannot be empty")
		}
	}

	return nil
}
