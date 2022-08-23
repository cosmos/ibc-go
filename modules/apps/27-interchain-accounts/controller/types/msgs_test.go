package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

var (
	testAccAddress     = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"
	testMetadataString = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
)

func TestMsgRegisterAccountValidateBasic(t *testing.T) {
	var msg *types.MsgRegisterAccount

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"connection id is invalid",
			func() {
				msg.ConnectionId = ""
			},
			false,
		},
		{
			"owner address is empty",
			func() {
				msg.Owner = ""
			},
			false,
		},
		{
			"owner address is invalid",
			func() {
				msg.Owner = "invalid_address"
			},
			false,
		},
	}

	for i, tc := range testCases {

		msg = types.NewMsgRegisterAccount(ibctesting.FirstConnectionID, testAccAddress, testMetadataString)

		tc.malleate()

		err := msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func TestMsgRegisterAccountGetSigners(t *testing.T) {
	expSigner, err := sdk.AccAddressFromBech32(testAccAddress)
	require.NoError(t, err)

	msg := types.NewMsgRegisterAccount(ibctesting.FirstConnectionID, testAccAddress, testMetadataString)
	require.Equal(t, []sdk.AccAddress{expSigner}, msg.GetSigners())
}
