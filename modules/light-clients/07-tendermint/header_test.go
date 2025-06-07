package tendermint_test

import (
	"errors"
	"time"

	cmtprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func (s *TendermintTestSuite) TestGetHeight() {
	header := s.chainA.LatestCommittedHeader
	s.Require().NotEqual(uint64(0), header.GetHeight())
}

func (s *TendermintTestSuite) TestGetTime() {
	header := s.chainA.LatestCommittedHeader
	s.Require().NotEqual(time.Time{}, header.GetTime())
}

func (s *TendermintTestSuite) TestHeaderValidateBasic() {
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
			header = s.chainA.LatestCommittedHeader
			header.Commit = nil
		}, errors.New("header failed basic validation")},
		{"trusted height is equal to header height", func() {
			var ok bool
			header.TrustedHeight, ok = header.GetHeight().(clienttypes.Height)
			s.Require().True(ok)
		}, errors.New("invalid header height")},
		{"validator set nil", func() {
			header.ValidatorSet = nil
		}, errors.New("invalid client header")},
		{"ValidatorSetFromProto failed", func() {
			header.ValidatorSet.Validators[0].PubKey = cmtprotocrypto.PublicKey{}
		}, errors.New("validator set is not tendermint validator set")},
		{"header validator hash does not equal hash of validator set", func() {
			// use chainB's randomly generated validator set
			header.ValidatorSet = s.chainB.LatestCommittedHeader.ValidatorSet
		}, errors.New("validator set does not match hash")},
	}

	s.Require().Equal(exported.Tendermint, s.header.ClientType())

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			header = s.chainA.LatestCommittedHeader // must be explicitly changed in malleate

			tc.malleate()

			err := header.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
