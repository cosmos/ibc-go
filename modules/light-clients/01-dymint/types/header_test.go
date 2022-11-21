package types_test

import (
	"time"

	tmprotocrypto "github.com/tendermint/tendermint/proto/tendermint/crypto"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/01-dymint/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *DymintTestSuite) TestGetHeight() {
	header := suite.chainA.TestChainClient.(*ibctesting.TestChainDymint).LastHeader
	suite.Require().NotEqual(uint64(0), header.GetHeight())
}

func (suite *DymintTestSuite) TestGetTime() {
	header := suite.chainA.TestChainClient.(*ibctesting.TestChainDymint).LastHeader
	suite.Require().NotEqual(time.Time{}, header.GetTime())
}

func (suite *DymintTestSuite) TestHeaderValidateBasic() {
	var (
		header *types.Header
	)
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{"valid header", func() {}, true},
		{"header is nil", func() {
			header.Header = nil
		}, false},
		{"signed header is nil", func() {
			header.SignedHeader = nil
		}, false},
		{"SignedHeaderFromProto failed", func() {
			header.SignedHeader.Commit.Height = -1
		}, false},
		{"signed header failed dymint ValidateBasic", func() {
			header = suite.chainA.TestChainClient.(*ibctesting.TestChainDymint).LastHeader
			header.SignedHeader.Commit = nil
		}, false},
		{"trusted height is equal to header height", func() {
			header.TrustedHeight = header.GetHeight().(clienttypes.Height)
		}, false},
		{"validator set nil", func() {
			header.ValidatorSet = nil
		}, false},
		{"ValidatorSetFromProto failed", func() {
			header.ValidatorSet.Validators[0].PubKey = tmprotocrypto.PublicKey{}
		}, false},
		{"header validator hash does not equal hash of validator set", func() {
			// use chainB's randomly generated validator set
			header.ValidatorSet = suite.chainB.TestChainClient.(*ibctesting.TestChainDymint).LastHeader.ValidatorSet
		}, false},
	}

	suite.Require().Equal(exported.Dymint, suite.header.ClientType())

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			header = suite.chainA.TestChainClient.(*ibctesting.TestChainDymint).LastHeader // must be explicitly changed in malleate

			tc.malleate()

			err := header.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
