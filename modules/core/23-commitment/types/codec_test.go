package types_test

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
)

func (s *MerkleTestSuite) TestCodecTypeRegistration() {
	testCases := []struct {
		name    string
		typeURL string
		expErr  error
	}{
		{
			"success: MerkleRoot",
			sdk.MsgTypeURL(&types.MerkleRoot{}),
			nil,
		},
		{
			"success: MerklePrefix",
			sdk.MsgTypeURL(&types.MerklePrefix{}),
			nil,
		},
		{
			"success: MerklePath",
			sdk.MsgTypeURL(&v2.MerklePath{}),
			nil,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			errors.New("unable to resolve type URL ibc.invalid.MsgTypeURL"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expErr == nil {
				s.Require().NotNil(msg)
				s.Require().NoError(err)
			} else {
				s.Nil(msg)
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
