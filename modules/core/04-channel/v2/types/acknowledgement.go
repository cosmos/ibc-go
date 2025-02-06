package types

import (
	"crypto/sha256"

	errorsmod "cosmossdk.io/errors"
)

var ErrorAcknowledgement = sha256.Sum256([]byte("UNIVERSAL ERROR ACKNOWLEDGEMENT"))

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
