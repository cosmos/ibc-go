package solomachine_test

import (
	"errors"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *SoloMachineTestSuite) TestHeaderValidateBasic() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		header := sm.CreateHeader(sm.Diversifier)

		cases := []struct {
			name   string
			header *solomachine.Header
			expErr error
		}{
			{
				"valid header",
				header,
				nil,
			},
			{
				"timestamp is zero",
				&solomachine.Header{
					Timestamp:      0,
					Signature:      header.Signature,
					NewPublicKey:   header.NewPublicKey,
					NewDiversifier: header.NewDiversifier,
				},
				errors.New("invalid timestamp, it must represent a valid time greater than zero"),
			},
			{
				"signature is empty",
				&solomachine.Header{
					Timestamp:      header.Timestamp,
					Signature:      []byte{},
					NewPublicKey:   header.NewPublicKey,
					NewDiversifier: header.NewDiversifier,
				},
				errors.New("the signature is empty, which is invalid"),
			},
			{
				"diversifier contains only spaces",
				&solomachine.Header{
					Timestamp:      header.Timestamp,
					Signature:      header.Signature,
					NewPublicKey:   header.NewPublicKey,
					NewDiversifier: " ",
				},
				errors.New("the diversifier contains only whitespace, which is invalid"),
			},
			{
				"public key is nil",
				&solomachine.Header{
					Timestamp:      header.Timestamp,
					Signature:      header.Signature,
					NewPublicKey:   nil,
					NewDiversifier: header.NewDiversifier,
				},
				errors.New("the public key is nil, which is invalid"),
			},
		}

		suite.Require().Equal(exported.Solomachine, header.ClientType())

		for _, tc := range cases {
			tc := tc

			suite.Run(tc.name, func() {
				err := tc.header.ValidateBasic()

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}
