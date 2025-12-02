package attestations_test

import (
	attestations "github.com/cosmos/ibc-go/v10/modules/light-clients/10-attestations"
)

func (s *AttestationsTestSuite) TestAttestationProofValidateBasic() {
	testCases := []struct {
		name             string
		attestationProof attestations.AttestationProof
		expErr           string
	}{
		{
			name: "valid proof",
			attestationProof: attestations.AttestationProof{
				AttestationData: []byte("valid data"),
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
			name: "empty signatures",
			attestationProof: attestations.AttestationProof{
				AttestationData: []byte("valid data"),
				Signatures:      [][]byte{},
			},
			expErr: "signatures cannot be empty",
		},
		{
			name: "invalid signature length",
			attestationProof: attestations.AttestationProof{
				AttestationData: []byte("valid data"),
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
