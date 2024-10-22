package types_test

import (
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (s *TypesTestSuite) TestValidateChannel() {
	var c types.Channel
	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "failure: invalid ClientID",
			malleate: func() {
				c.ClientId = ""
			},
			expErr: host.ErrInvalidID,
		},
		{
			name: "failure: invalid counterparty channel id",
			malleate: func() {
				c.CounterpartyChannelId = ""
			},
			expErr: host.ErrInvalidID,
		},
		{
			name: "failure: invalid Merkle Path Prefix",
			malleate: func() {
				c.MerklePathPrefix.KeyPath = [][]byte{}
			},
			expErr: types.ErrInvalidChannel,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			c = types.NewChannel(ibctesting.FirstClientID, ibctesting.SecondClientID, ibctesting.MerklePath)

			tc.malleate()

			err := c.Validate()

			expPass := tc.expErr == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expErr)
			}
		})
	}
}
