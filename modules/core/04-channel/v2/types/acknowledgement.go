package types

import (
	"bytes"
	"crypto/sha256"

	proto "github.com/cosmos/gogoproto/proto"

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
