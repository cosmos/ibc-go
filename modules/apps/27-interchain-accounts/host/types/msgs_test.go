package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
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
					Authority: ibctesting.TestAccAddress,
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
					Authority: ibctesting.TestAccAddress,
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
	testCases := []struct {
		name    string
		address sdk.AccAddress
		expPass bool
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), true},
		{"failure: nil address", nil, false},
	}

	for _, tc := range testCases {
		msg := types.MsgUpdateParams{
			Authority: tc.address.String(),
			Params:    types.DefaultParams(),
		}
		if tc.expPass {
			require.Equal(t, []sdk.AccAddress{tc.address}, msg.GetSigners())
		} else {
			require.Panics(t, func() {
				msg.GetSigners()
			})
		}
	}
}
