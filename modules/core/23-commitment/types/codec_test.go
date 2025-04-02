package types_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
)

func (suite *MerkleTestSuite) TestCodecTypeRegistration() {
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
			fmt.Errorf("unable to resolve type URL ibc.invalid.MsgTypeURL"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expErr == nil {
				suite.NotNil(msg)
				suite.Require().NoError(err)
			} else {
				suite.Nil(msg)
				suite.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
