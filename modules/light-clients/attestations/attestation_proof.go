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
// Attestation data can be either a StateAttestation (for client updates) or
// a PacketAttestation (for packet membership/non-membership proofs).
func (ap AttestationProof) ValidateBasic() error {
	if len(ap.AttestationData) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "attestation data cannot be empty")
	}

	// Try to decode as PacketAttestation first (used for membership/non-membership proofs)
	packetAttestation, packetErr := ABIDecodePacketAttestation(ap.AttestationData)
	if packetErr == nil {
		if len(packetAttestation.Packets) == 0 {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "packets cannot be empty")
		}
	} else {
		// If that fails, try to decode as StateAttestation (used for client updates)
		if _, stateErr := ABIDecodeStateAttestation(ap.AttestationData); stateErr != nil {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "attestation data must be a valid StateAttestation or PacketAttestation")
		}
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
