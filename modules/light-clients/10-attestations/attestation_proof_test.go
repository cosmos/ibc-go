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
			"valid proof",
			attestations.AttestationProof{
				AttestationData: []byte("valid data"),
				Signatures:      [][]byte{make([]byte, 65)},
			},
			"",
		},
		{
			"empty attestation data",
			attestations.AttestationProof{
				AttestationData: []byte{},
				Signatures:      [][]byte{make([]byte, 65)},
			},
			"attestation data cannot be empty",
		},
		{
			"empty signatures",
			attestations.AttestationProof{
				AttestationData: []byte("valid data"),
				Signatures:      [][]byte{},
			},
			"signatures cannot be empty",
		},
		{
			"invalid signature length",
			attestations.AttestationProof{
				AttestationData: []byte("valid data"),
				Signatures:      [][]byte{make([]byte, 64)},
			},
			"signature 0 has invalid length",
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
