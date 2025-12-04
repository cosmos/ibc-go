package attestations_test

import (
	attestations "github.com/cosmos/ibc-go/v10/modules/light-clients/attestations"
)

func (s *AttestationsTestSuite) TestAttestationProofValidateBasic() {
	validPacketAttestation := &attestations.PacketAttestation{
		Height: 100,
		Packets: []attestations.PacketCompact{
			{Path: []byte("test-path"), Commitment: []byte("test-commitment")},
		},
	}
	validPacketAttestationData, err := validPacketAttestation.ABIEncode()
	s.Require().NoError(err)

	validStateAttestation := &attestations.StateAttestation{
		Height:    100,
		Timestamp: 1234567890000000000,
	}
	validStateAttestationData, err := validStateAttestation.ABIEncode()
	s.Require().NoError(err)

	emptyPacketAttestation := &attestations.PacketAttestation{
		Height:  100,
		Packets: []attestations.PacketCompact{},
	}
	emptyPacketsData, err := emptyPacketAttestation.ABIEncode()
	s.Require().NoError(err)

	testCases := []struct {
		name             string
		attestationProof attestations.AttestationProof
		expErr           string
	}{
		{
			name: "valid packet attestation proof",
			attestationProof: attestations.AttestationProof{
				AttestationData: validPacketAttestationData,
				Signatures:      [][]byte{make([]byte, 65)},
			},
			expErr: "",
		},
		{
			name: "valid state attestation proof",
			attestationProof: attestations.AttestationProof{
				AttestationData: validStateAttestationData,
				Signatures:      [][]byte{make([]byte, 65)},
			},
			expErr: "",
		},
		{
			name: "empty attestation data",
			attestationProof: attestations.AttestationProof{
				AttestationData: []byte{},
				Signatures:      [][]byte{make([]byte, 65)},
			},
			expErr: "attestation data cannot be empty",
		},
		{
			name: "invalid attestation data",
			attestationProof: attestations.AttestationProof{
				AttestationData: []byte("invalid data"),
				Signatures:      [][]byte{make([]byte, 65)},
			},
			expErr: "attestation data must be a valid StateAttestation or PacketAttestation",
		},
		{
			name: "empty packets in packet attestation",
			attestationProof: attestations.AttestationProof{
				AttestationData: emptyPacketsData,
				Signatures:      [][]byte{make([]byte, 65)},
			},
			expErr: "packets cannot be empty",
		},
		{
			name: "empty signatures",
			attestationProof: attestations.AttestationProof{
				AttestationData: validPacketAttestationData,
				Signatures:      [][]byte{},
			},
			expErr: "signatures cannot be empty",
		},
		{
			name: "invalid signature length",
			attestationProof: attestations.AttestationProof{
				AttestationData: validPacketAttestationData,
				Signatures:      [][]byte{make([]byte, 64)},
			},
			expErr: "signature 0 has invalid length",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.attestationProof.ValidateBasic()
			if tc.expErr != "" {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expErr)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
