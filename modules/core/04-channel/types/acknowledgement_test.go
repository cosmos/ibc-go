package types_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	abci "github.com/cometbft/cometbft/api/cometbft/abci/v1"
	cmtstate "github.com/cometbft/cometbft/state"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

const (
	gasUsed   = uint64(100)
	gasWanted = uint64(100)
)

// tests acknowledgement.ValidateBasic and acknowledgement.Acknowledgement
func (suite *TypesTestSuite) TestAcknowledgement() {
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

// The safety of including ABCI error codes in the acknowledgement rests
// on the inclusion of these ABCI error codes in the abcitypes.ResponseDeliverTx
// hash. If the ABCI codes get removed from consensus they must no longer be used
// in the packet acknowledgement.
//
// This test acts as an indicator that the ABCI error codes may no longer be deterministic.
func (suite *TypesTestSuite) TestABCICodeDeterminism() {
	// same ABCI error code used
	err := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 1")
	errSameABCICode := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 2")

	// different ABCI error code used
	errDifferentABCICode := ibcerrors.ErrNotFound

	deliverTx := responseExecTxResultWithEvents(err, gasUsed, gasWanted, []abci.Event{}, false)
	execTxResults := []*abci.ExecTxResult{deliverTx}

	deliverTxSameABCICode := responseExecTxResultWithEvents(errSameABCICode, gasUsed, gasWanted, []abci.Event{}, false)
	resultsSameABCICode := []*abci.ExecTxResult{deliverTxSameABCICode}

	deliverTxDifferentABCICode := responseExecTxResultWithEvents(errDifferentABCICode, gasUsed, gasWanted, []abci.Event{}, false)
	resultsDifferentABCICode := []*abci.ExecTxResult{deliverTxDifferentABCICode}

	hash := cmtstate.TxResultsHash(execTxResults)
	hashSameABCICode := cmtstate.TxResultsHash(resultsSameABCICode)
	hashDifferentABCICode := cmtstate.TxResultsHash(resultsDifferentABCICode)

	suite.Require().Equal(hash, hashSameABCICode)
	suite.Require().NotEqual(hash, hashDifferentABCICode)
}

// responseExecTxResultWithEvents returns an ABCI ExecTxResult object with fields
// filled in from the given error, gas values and events.
func responseExecTxResultWithEvents(err error, gw, gu uint64, events []abci.Event, debug bool) *abci.ExecTxResult {
	space, code, log := errorsmod.ABCIInfo(err, debug)
	return &abci.ExecTxResult{
		Codespace: space,
		Code:      code,
		Log:       log,
		GasWanted: int64(gw),
		GasUsed:   int64(gu),
		Events:    events,
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

func (suite *TypesTestSuite) TestAcknowledgementWithCodespace() {
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
