package attestations

import (
	"crypto/sha256"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	errorsmod "cosmossdk.io/errors"
)

const (
	// SignatureLength is the expected length of an ECDSA signature (r||s||v)
	SignatureLength = 65
)

// verifySignatures verifies that the attestation proof has valid signatures from unique attestors
// meeting the quorum threshold. Signatures cover sha256(attestationData).
func (cs *ClientState) verifySignatures(proof *AttestationProof) error {
	if len(proof.Signatures) == 0 {
		return errorsmod.Wrap(ErrInvalidSignature, "signatures cannot be empty")
	}

	attestorSet := make(map[string]bool)
	for _, addr := range cs.AttestorAddresses {
		attestorSet[strings.ToLower(addr)] = true
	}

	hash := sha256.Sum256(proof.AttestationData)
	seenSigners := make(map[string]bool)
	validSigs := 0

	for i, sig := range proof.Signatures {
		if len(sig) != SignatureLength {
			return errorsmod.Wrapf(ErrInvalidSignature, "signature %d has invalid length: expected %d, got %d", i, SignatureLength, len(sig))
		}

		recoveredPubKey, err := crypto.SigToPub(hash[:], sig)
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidSignature, "failed to recover public key from signature %d: %v", i, err)
		}

		recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)
		addrStr := strings.ToLower(recoveredAddr.Hex())

		if seenSigners[addrStr] {
			return errorsmod.Wrapf(ErrDuplicateSigner, "duplicate signer: %s", addrStr)
		}
		seenSigners[addrStr] = true

		if !attestorSet[addrStr] {
			return errorsmod.Wrapf(ErrUnknownSigner, "signer %s is not in attestor set", addrStr)
		}

		validSigs++
	}

	if validSigs < int(cs.MinRequiredSigs) {
		return errorsmod.Wrapf(ErrInvalidQuorum, "quorum not met: required %d, got %d", cs.MinRequiredSigs, validSigs)
	}

	return nil
}
