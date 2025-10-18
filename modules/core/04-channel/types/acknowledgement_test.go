package types_test

import (
	"errors"

	errorsmod "cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtstate "github.com/cometbft/cometbft/state"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

const (
	gasUsed   = uint64(100)
	gasWanted = uint64(100)
)

// tests acknowledgement.ValidateBasic and acknowledgement.Acknowledgement
func (s *TypesTestSuite) TestAcknowledgement() { //nolint:govet // this is a test, we are okay with copying locks
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
			types.NewErrorAcknowledgement(errors.New("error")),
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
			types.NewErrorAcknowledgement(errors.New("  ")),
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
		s.Run(tc.name, func() {
			s.SetupTest()

			err := tc.ack.ValidateBasic()

			if tc.expValidates {
				s.Require().NoError(err)

				// expect all valid acks to be able to be marshaled
				s.Require().NotPanics(func() {
					bz := tc.ack.Acknowledgement()
					s.Require().NotNil(bz)
					s.Require().Equal(tc.expBytes, bz)
				})
			} else {
				s.Require().Error(err)
			}

			s.Require().Equal(tc.expSuccess, tc.ack.Success())
		})
	}
}

// The safety of including ABCI error codes in the acknowledgement rests
// on the inclusion of these ABCI error codes in the abcitypes.ResponseDeliverTx
// hash. If the ABCI codes get removed from consensus they must no longer be used
// in the packet acknowledgement.
//
// This test acts as an indicator that the ABCI error codes may no longer be deterministic.
func (s *TypesTestSuite) TestABCICodeDeterminism() {
	// same ABCI error code used
	err := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 1")
	errSameABCICode := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 2")

	// different ABCI error code used
	errDifferentABCICode := ibcerrors.ErrNotFound

	deliverTx := sdkerrors.ResponseExecTxResultWithEvents(err, gasUsed, gasWanted, []abcitypes.Event{}, false)
	execTxResults := []*abcitypes.ExecTxResult{deliverTx}

	deliverTxSameABCICode := sdkerrors.ResponseExecTxResultWithEvents(errSameABCICode, gasUsed, gasWanted, []abcitypes.Event{}, false)
	resultsSameABCICode := []*abcitypes.ExecTxResult{deliverTxSameABCICode}

	deliverTxDifferentABCICode := sdkerrors.ResponseExecTxResultWithEvents(errDifferentABCICode, gasUsed, gasWanted, []abcitypes.Event{}, false)
	resultsDifferentABCICode := []*abcitypes.ExecTxResult{deliverTxDifferentABCICode}

	hash := cmtstate.TxResultsHash(execTxResults)
	hashSameABCICode := cmtstate.TxResultsHash(resultsSameABCICode)
	hashDifferentABCICode := cmtstate.TxResultsHash(resultsDifferentABCICode)

	s.Require().Equal(hash, hashSameABCICode)
	s.Require().NotEqual(hash, hashDifferentABCICode)
}

// TestAcknowledgementError will verify that only a constant string and
// ABCI error code are used in constructing the acknowledgement error string
func (s *TypesTestSuite) TestAcknowledgementError() {
	// same ABCI error code used
	err := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 1")
	errSameABCICode := errorsmod.Wrap(ibcerrors.ErrOutOfGas, "error string 2")

	// different ABCI error code used
	errDifferentABCICode := ibcerrors.ErrNotFound

	ack := types.NewErrorAcknowledgement(err)
	ackSameABCICode := types.NewErrorAcknowledgement(errSameABCICode)
	ackDifferentABCICode := types.NewErrorAcknowledgement(errDifferentABCICode)

	s.Require().Equal(ack, ackSameABCICode)
	s.Require().NotEqual(ack, ackDifferentABCICode)
}

func (s *TypesTestSuite) TestAcknowledgementWithCodespace() { //nolint:govet // this is a test, we are okay with copying locks
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
			types.NewErrorAcknowledgementWithCodespace(errors.New("unknown error")),
			[]byte(`{"error":"ABCI error: undefined/1: error handling packet: see events for details"}`),
		},
		{
			"nil error",
			types.NewErrorAcknowledgementWithCodespace(nil),
			[]byte(`{"error":"ABCI error: /0: error handling packet: see events for details"}`),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.expBytes, tc.ack.Acknowledgement())
		})
	}
}
