package types

import (
	"bytes"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

var _ exported.Header = &ConflictingSignaturesHeader{}

// ClientType is a Solo Machine light client.
func (h ConflictingSignaturesHeader) ClientType() string {
	return exported.Solomachine
}

// GetClientID returns the ID of the client that committed a ConflictingSignaturesHeader.
func (h ConflictingSignaturesHeader) GetClientID() string {
	return h.ClientId
}

// GetHeight is added for compatibility between PRs.
// TODO: Remove after GetHeight() is removed from Header interface.
func (h ConflictingSignaturesHeader) GetHeight() exported.Height {
	return nil
}

// Type implements Evidence interface.
func (h ConflictingSignaturesHeader) Type() string {
	return exported.TypeClientMisbehaviour
}

// ValidateBasic implements Evidence interface.
func (h ConflictingSignaturesHeader) ValidateBasic() error {
	if err := host.ClientIdentifierValidator(h.ClientId); err != nil {
		return sdkerrors.Wrap(err, "invalid client identifier for solo machine")
	}

	if h.Sequence == 0 {
		return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "sequence cannot be 0")
	}

	if err := h.SignatureOne.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "signature one failed basic validation")
	}

	if err := h.SignatureTwo.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "signature two failed basic validation")
	}

	// ConflictingSignaturesHeader signatures cannot be identical
	if bytes.Equal(h.SignatureOne.Signature, h.SignatureTwo.Signature) {
		return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "ConflictingSignaturesHeader signatures cannot be equal")
	}

	// message data signed cannot be identical
	if bytes.Equal(h.SignatureOne.Data, h.SignatureTwo.Data) {
		return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "ConflictingSignaturesHeader signature data must be signed over different messages")
	}

	return nil
}

// ValidateBasic ensures that the signature and data fields are non-empty.
func (sd SignatureAndData) ValidateBasic() error {
	if len(sd.Signature) == 0 {
		return sdkerrors.Wrap(ErrInvalidSignatureAndData, "signature cannot be empty")
	}
	if len(sd.Data) == 0 {
		return sdkerrors.Wrap(ErrInvalidSignatureAndData, "data for signature cannot be empty")
	}
	if sd.DataType == UNSPECIFIED {
		return sdkerrors.Wrap(ErrInvalidSignatureAndData, "data type cannot be UNSPECIFIED")
	}
	if sd.Timestamp == 0 {
		return sdkerrors.Wrap(ErrInvalidSignatureAndData, "timestamp cannot be 0")
	}

	return nil
}
