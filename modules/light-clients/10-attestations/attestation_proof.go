package attestations

import (
	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.ClientMessage = (*AttestationProof)(nil)

// ClientType defines that the AttestationProof is for Attestations.
func (AttestationProof) ClientType() string {
	return exported.Attestations
}

// ValidateBasic ensures that the attestation data and signatures are initialized.
func (ap AttestationProof) ValidateBasic() error {
	packetAttestation, err := ABIDecodePacketAttestation(ap.AttestationData)
	if err != nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, err.Error())
	}

	if len(packetAttestation.Packets) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "packets cannot be empty")
	}

	if len(ap.Signatures) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "signatures cannot be empty")
	}

	for i, sig := range ap.Signatures {
		if len(sig) != SignatureLength {
			return errorsmod.Wrapf(clienttypes.ErrInvalidHeader, "signature %d has invalid length: expected %d, got %d", i, SignatureLength, len(sig))
		}
	}

	return nil
}
