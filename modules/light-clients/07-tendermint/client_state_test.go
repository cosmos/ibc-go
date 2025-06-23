package tendermint_test

import (
	ics23 "github.com/cosmos/ics23/go"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

const (
	// Do not change the length of these variables
	fiftyCharChainID    = "12345678901234567890123456789012345678901234567890"
	fiftyOneCharChainID = "123456789012345678901234567890123456789012345678901"
)

var invalidProof = []byte("invalid proof")

func (s *TendermintTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *ibctm.ClientState
		expErr      error
	}{
		{
			name:        "valid client",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      nil,
		},
		{
			name:        "valid client with nil upgrade path",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), nil),
			expErr:      nil,
		},
		{
			name:        "invalid chainID",
			clientState: ibctm.NewClientState("  ", ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidChainID,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "valid chainID - chainID validation did not fail for chainID of length 50! ",
			clientState: ibctm.NewClientState(fiftyCharChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      nil,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "invalid chainID - chainID validation failed for chainID of length 51! ",
			clientState: ibctm.NewClientState(fiftyOneCharChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidChainID,
		},
		{
			name:        "invalid trust level",
			clientState: ibctm.NewClientState(chainID, ibctm.Fraction{Numerator: 0, Denominator: 1}, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidTrustLevel,
		},
		{
			name:        "invalid zero trusting period",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, 0, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidTrustingPeriod,
		},
		{
			name:        "invalid negative trusting period",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, -1, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidTrustingPeriod,
		},
		{
			name:        "invalid zero unbonding period",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, 0, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidUnbondingPeriod,
		},
		{
			name:        "invalid negative unbonding period",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, -1, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidUnbondingPeriod,
		},
		{
			name:        "invalid zero max clock drift",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, 0, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidMaxClockDrift,
		},
		{
			name:        "invalid negative max clock drift",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, -1, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidMaxClockDrift,
		},
		{
			name:        "invalid revision number",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidHeaderHeight,
		},
		{
			name:        "invalid revision height",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.ZeroHeight(), commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidHeaderHeight,
		},
		{
			name:        "trusting period not less than unbonding period",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, ubdPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibctm.ErrInvalidTrustingPeriod,
		},
		{
			name:        "proof specs is nil",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, nil, upgradePath),
			expErr:      ibctm.ErrInvalidProofSpecs,
		},
		{
			name:        "proof specs contains nil",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, []*ics23.ProofSpec{ics23.TendermintSpec, nil}, upgradePath),
			expErr:      ibctm.ErrInvalidProofSpecs,
		},
		{
			name:        "invalid upgrade path",
			clientState: ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), invalidUpgradePath),
			expErr:      clienttypes.ErrInvalidClient,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.clientState.Validate()

			if tc.expErr == nil {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
