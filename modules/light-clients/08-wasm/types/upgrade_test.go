package types_test

import (
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
)

// TestVerifyUpgrade currently only tests the interface into the contract.
// Test code is used in the grandpa contract.
// New client state, consensus state, and client metadata is expected to be set in the contract on success
func (suite *WasmTestSuite) TestVerifyUpgrade() {
	var (
		clientState            exported.ClientState
		upgradedClient         exported.ClientState
		upgradedConsState      exported.ConsensusState
		proofUpgradedClient    []byte
		proofUpgradedConsState []byte
		err                    error
		ok                     bool
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful upgrade",
			func() {},
			true,
		},
		{
			"unsuccessful upgrade: invalid new client state",
			func() {
				upgradedClient = &solomachine.ClientState{}
			},
			false,
		},
		{
			"unsuccessful upgrade: invalid new consensus state",
			func() {
				upgradedConsState = &solomachine.ConsensusState{}
			},
			false,
		},
		{
			"unsuccessful upgrade: invalid client state proof",
			func() {
				proofUpgradedClient = []byte("invalid client state proof")
			},
			false,
		},
		{
			"unsuccessful upgrade: invalid consensus state proof",
			func() {
				proofUpgradedConsState = []byte("invalid consensus state proof")
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		// reset suite
		suite.SetupWithChannel()
		clientState = suite.clientState
		upgradedClient = suite.clientState
		upgradedConsState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetLatestClientConsensusState(suite.ctx, "08-wasm-0")
		suite.Require().True(ok)
		proofUpgradedClient = []byte("upgraded client state proof")
		proofUpgradedConsState = []byte("upgraded consensus state proof")

		tc.setup()

		err = clientState.VerifyUpgradeAndUpdateState(
			suite.ctx,
			suite.chainA.Codec,
			suite.store,
			upgradedClient,
			upgradedConsState,
			proofUpgradedClient,
			proofUpgradedConsState,
		)

		if tc.expPass {
			suite.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)
		} else {
			suite.Require().Error(err, "verify upgrade passed on invalid case: %s", tc.name)
		}
	}
}
