package solomachine_test

import (
	"errors"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
				errors.New("timestamp cannot be zero: invalid client header"),
			},
			{
				"signature is empty",
				&solomachine.Header{
					Timestamp:      header.Timestamp,
					Signature:      []byte{},
					NewPublicKey:   header.NewPublicKey,
					NewDiversifier: header.NewDiversifier,
				},
				errors.New("signature cannot be empty: invalid client header"),
			},
			{
				"diversifier contains only spaces",
				&solomachine.Header{
					Timestamp:      header.Timestamp,
					Signature:      header.Signature,
					NewPublicKey:   header.NewPublicKey,
					NewDiversifier: " ",
				},
				errors.New("diversifier cannot contain only spaces: invalid client header"),
			},
			{
				"public key is nil",
				&solomachine.Header{
					Timestamp:      header.Timestamp,
					Signature:      header.Signature,
					NewPublicKey:   nil,
					NewDiversifier: header.NewDiversifier,
				},
				errors.New("new public key cannot be empty: invalid client header"),
			},
		}

		suite.Require().Equal(exported.Solomachine, header.ClientType())

		for _, tc := range cases {
			suite.Run(tc.name, func() {
				err := tc.header.ValidateBasic()

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}
