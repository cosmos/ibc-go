package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	"github.com/stretchr/testify/require"
)

func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	var msg *types.MsgUpdateParams

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: valid authority address",
			func() {
				msg = &types.MsgUpdateParams{
					Authority: sdk.AccAddress("authority").String(),
					Params:    types.DefaultParams(),
				}
			},
			true,
		},
		{
			"failure: invalid authority address",
			func() {
				msg = &types.MsgUpdateParams{
					Authority: "authority",
				}
			},
			false,
		},
		{
			"failure: invalid allowed message",
			func() {
				msg = &types.MsgUpdateParams{
					Authority: sdk.AccAddress("authority").String(),
					Params: types.Params{
						AllowMessages: []string{""},
					},
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.malleate()

			err := msg.ValidateBasic()
			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMsgUpdateParamsGetSigners(t *testing.T) {
	authority := sdk.AccAddress("authority")
	msg := types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    types.DefaultParams(),
	}
	require.Equal(t, []sdk.AccAddress{authority}, msg.GetSigners())
}
