package attestations

import (
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	errorsmod "cosmossdk.io/errors"
)

const (
	// SignatureLength is the expected length of an ECDSA signature (r||s||v)
	SignatureLength = 65
	// recoveryIDIndex is the byte position of the recovery ID (v) in the signature
	recoveryIDIndex = 64
)

// verifySignatures verifies that the attestation proof has valid signatures from unique attestors
// meeting the quorum threshold. Signatures cover sha256(attestationData).
func (cs *ClientState) verifySignatures(proof *AttestationProof) error {
	if len(proof.Signatures) == 0 {
		return errorsmod.Wrap(ErrInvalidSignature, "signatures cannot be empty")
	}

	attestorSet := make(map[common.Address]bool)
	for _, addr := range cs.AttestorAddresses {
		attestorSet[common.HexToAddress(addr)] = true
	}

	hash := sha256.Sum256(proof.AttestationData)
	seenSigners := make(map[common.Address]bool)

	for i, sig := range proof.Signatures {
		if len(sig) != SignatureLength {
			return errorsmod.Wrapf(ErrInvalidSignature, "signature %d has invalid length: expected %d, got %d", i, SignatureLength, len(sig))
		}

		normalizedSig := normalizeSignature(sig)

		recoveredPubKey, err := crypto.SigToPub(hash[:], normalizedSig)
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidSignature, "failed to recover public key from signature %d: %v", i, err)
		}

		if recoveredPubKey == nil {
			return errorsmod.Wrapf(ErrInvalidSignature, "recovered public key is nil for signature %d", i)
		}

		recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)

		if seenSigners[recoveredAddr] {
			return errorsmod.Wrapf(ErrDuplicateSigner, "duplicate signer: %s", recoveredAddr.Hex())
		}
		seenSigners[recoveredAddr] = true

		if !attestorSet[recoveredAddr] {
			return errorsmod.Wrapf(ErrUnknownSigner, "signer %s is not in attestor set", recoveredAddr.Hex())
		}
	}

	if len(proof.Signatures) < int(cs.MinRequiredSigs) {
		return errorsmod.Wrapf(ErrInvalidQuorum, "quorum not met: required %d, got %d", cs.MinRequiredSigs, len(proof.Signatures))
	}

	return nil
}

// normalizeSignature converts the ECDSA recovery ID (v) from Ethereum format (27/28)
// to raw format (0/1). go-ethereum's crypto.SigToPub expects raw format, while
// Solidity's ECDSA.recover and most signing libraries produce Ethereum format.
func normalizeSignature(sig []byte) []byte {
	normalized := make([]byte, SignatureLength)
	copy(normalized, sig)

	v := normalized[recoveryIDIndex]
	switch v {
	case 27:
		normalized[recoveryIDIndex] = 0
	case 28:
		normalized[recoveryIDIndex] = 1
	default:
		// Already in raw format (0/1) or unknown, leave unchanged
	}

	return normalized
}
