package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func (s *TypesTestSuite) TestMarshalHeader() {
	cdc := s.chainA.App.AppCodec()
	h := &ibctm.Header{
		TrustedHeight: types.NewHeight(4, 100),
	}

	// marshal header
	bz, err := types.MarshalClientMessage(cdc, h)
	s.Require().NoError(err)

	// unmarshal header
	newHeader, err := types.UnmarshalClientMessage(cdc, bz)
	s.Require().NoError(err)

	s.Require().Equal(h, newHeader)

	// use invalid bytes
	invalidHeader, err := types.UnmarshalClientMessage(cdc, []byte("invalid bytes"))
	s.Require().Error(err)
	s.Require().Nil(invalidHeader)
}
