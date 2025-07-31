package types

import (
	"bytes"
	"crypto/sha256"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	ErrorAcknowledgement                          = sha256.Sum256([]byte("UNIVERSAL_ERROR_ACKNOWLEDGEMENT"))
	_                    exported.Acknowledgement = &Acknowledgement{}
)

// NewAcknowledgement creates a new Acknowledgement containing the provided app acknowledgements.
func NewAcknowledgement(appAcknowledgements ...[]byte) Acknowledgement {
	return Acknowledgement{AppAcknowledgements: appAcknowledgements}
}

// Validate performs a basic validation of the acknowledgement
func (ack Acknowledgement) Validate() error {
	// acknowledgement list should be non-empty
	if len(ack.AppAcknowledgements) == 0 {
		return errorsmod.Wrap(ErrInvalidAcknowledgement, "app acknowledgements must be non-empty")
	}

	for _, a := range ack.AppAcknowledgements {
		// Each app acknowledgement should be non-empty
		if len(a) == 0 {
			return errorsmod.Wrap(ErrInvalidAcknowledgement, "app acknowledgement cannot be empty")
		}

		// Ensure that the app acknowledgement contains ErrorAcknowledgement
		// **if and only if** the app acknowledgement list has a single element
		if len(ack.AppAcknowledgements) > 1 {
			if bytes.Equal(a, ErrorAcknowledgement[:]) {
				return errorsmod.Wrap(ErrInvalidAcknowledgement, "cannot have the error acknowledgement in multi acknowledgement list")
			}
		}
	}

	return nil
}

// Success returns true if the acknowledgement is successful
// it implements the exported.Acknowledgement interface
func (ack Acknowledgement) Success() bool {
	return !bytes.Equal(ack.AppAcknowledgements[0], ErrorAcknowledgement[:])
}

// Acknowledgement returns the acknowledgement bytes to implement the acknowledgement interface
func (ack Acknowledgement) Acknowledgement() []byte {
	bz, err := proto.Marshal(&ack)
	if err != nil {
		panic(err)
	}
	return bz
}
