package types_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// tests acknowledgement.ValidateBasic and acknowledgement.Acknowledgement
func (suite TypesTestSuite) TestAcknowledgement() { //nolint:govet // this is a test, we are okay with copying locks
	testCases := []struct {
		name         string
		ack          types.Acknowledgement
		expValidates bool
		expBytes     []byte
		expSuccess   bool // indicate if this is a success or failed ack
	}{
		{
			"valid successful ack",
			types.NewResultAcknowledgement([]byte("success")),
			true,
			[]byte(`{"result":"c3VjY2Vzcw=="}`),
			true,
		},
		{
			"valid failed ack",
			types.NewErrorAcknowledgement(fmt.Errorf("error")),
			true,
			[]byte(`{"error":"ABCI code: 1: error handling packet: see events for details"}`),
			false,
		},
		{
			"empty successful ack",
			types.NewResultAcknowledgement([]byte{}),
			false,
			nil,
			true,
		},
		{
			"empty failed ack",
			types.NewErrorAcknowledgement(fmt.Errorf("  ")),
			true,
			[]byte(`{"error":"ABCI code: 1: error handling packet: see events for details"}`),
			false,
		},
		{
			"nil response",
			types.Acknowledgement{
				Response: nil,
			},
			false,
			nil,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			err := tc.ack.ValidateBasic()

			if tc.expValidates {
				suite.Require().NoError(err)

				// expect all valid acks to be able to be marshaled
				suite.NotPanics(func() {
					bz := tc.ack.Acknowledgement()
					suite.Require().NotNil(bz)
					suite.Require().Equal(tc.expBytes, bz)
				})
			} else {
				suite.Require().Error(err)
			}

			suite.Require().Equal(tc.expSuccess, tc.ack.Success())
		})
	}
}

// TestAcknowledgementError will verify that only a constant string and
// ABCI error code are used in constructing the acknowledgement error string
func (suite *TypesTestSuite) TestAcknowledgementError() {
	// same ABCI error code used
	err := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 1")
	errSameABCICode := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 2")

	// different ABCI error code used
	errDifferentABCICode := ibcerrors.ErrNotFound

	ack := types.NewErrorAcknowledgement(err)
	ackSameABCICode := types.NewErrorAcknowledgement(errSameABCICode)
	ackDifferentABCICode := types.NewErrorAcknowledgement(errDifferentABCICode)

	suite.Require().Equal(ack, ackSameABCICode)
	suite.Require().NotEqual(ack, ackDifferentABCICode)
}

func (suite TypesTestSuite) TestAcknowledgementWithCodespace() { //nolint:govet // this is a test, we are okay with copying locks
	testCases := []struct {
		name     string
		ack      types.Acknowledgement
		expBytes []byte
	}{
		{
			"valid failed ack",
			types.NewErrorAcknowledgementWithCodespace(ibcerrors.ErrInsufficientFunds),
			[]byte(`{"error":"ABCI error: ibc/3: error handling packet: see events for details"}`),
		},
		{
			"unknown error",
			types.NewErrorAcknowledgementWithCodespace(fmt.Errorf("unknown error")),
			[]byte(`{"error":"ABCI error: undefined/1: error handling packet: see events for details"}`),
		},
		{
			"nil error",
			types.NewErrorAcknowledgementWithCodespace(nil),
			[]byte(`{"error":"ABCI error: /0: error handling packet: see events for details"}`),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expBytes, tc.ack.Acknowledgement())
		})
	}
}
