package tendermint_test

import (
	"errors"
	"time"

	cmtprotocrypto "github.com/cometbft/cometbft/api/cometbft/crypto/v1"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func (suite *TendermintTestSuite) TestGetHeight() {
	header := suite.chainA.LatestCommittedHeader
	suite.Require().NotEqual(uint64(0), header.GetHeight())
}

func (suite *TendermintTestSuite) TestGetTime() {
	header := suite.chainA.LatestCommittedHeader
	suite.Require().NotEqual(time.Time{}, header.GetTime())
}

func (suite *TendermintTestSuite) TestHeaderValidateBasic() {
	var header *ibctm.Header
	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"valid header", func() {}, nil},
		{"header is nil", func() {
			header.Header = nil
		}, errors.New("tendermint header cannot be nil")},
		{"signed header is nil", func() {
			header.SignedHeader = nil
		}, errors.New("tendermint signed header cannot be nil")},
		{"SignedHeaderFromProto failed", func() {
			header.Commit.Height = -1
		}, errors.New("header is not a tendermint header")},
		{"signed header failed tendermint ValidateBasic", func() {
			header = suite.chainA.LatestCommittedHeader
			header.Commit = nil
		}, errors.New("header failed basic validation")},
		{"trusted height is equal to header height", func() {
			var ok bool
			header.TrustedHeight, ok = header.GetHeight().(clienttypes.Height)
			suite.Require().True(ok)
		}, errors.New("invalid header height")},
		{"validator set nil", func() {
			header.ValidatorSet = nil
		}, errors.New("invalid client header")},
		{"ValidatorSetFromProto failed", func() {
			header.ValidatorSet.Validators[0].PubKeyType = ""
			header.ValidatorSet.Validators[0].PubKeyBytes = []byte{}
			header.ValidatorSet.Validators[0].PubKey = &cmtprotocrypto.PublicKey{}
		}, errors.New("validator set is not tendermint validator set")},
		{"header validator hash does not equal hash of validator set", func() {
			// use chainB's randomly generated validator set
			header.ValidatorSet = suite.chainB.LatestCommittedHeader.ValidatorSet
		}, errors.New("validator set does not match hash")},
	}

	suite.Require().Equal(exported.Tendermint, suite.header.ClientType())

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			header = suite.chainA.LatestCommittedHeader // must be explicitly changed in malleate

			tc.malleate()

			err := header.ValidateBasic()

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
