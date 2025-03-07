package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestMsgRegisterCounterpartyValidateBasic(t *testing.T) {
	signer := ibctesting.TestAccAddress
	testCases := []struct {
		name     string
		msg      *types.MsgRegisterCounterparty
		expError error
	}{
		{
			"success",
			types.NewMsgRegisterCounterparty(
				"testclientid-3",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-2",
				signer,
			),
			nil,
		},
		{
			"failure: client id does not match clientID format",
			types.NewMsgRegisterCounterparty(
				"testclientid1",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-3",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: counterparty client id does not match clientID format",
			types.NewMsgRegisterCounterparty(
				"testclientid-1",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid2",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: empty client id",
			types.NewMsgRegisterCounterparty(
				"",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-3",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: empty counterparty client id",
			types.NewMsgRegisterCounterparty(
				"testclientid-1",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: empty counterparty messaging key",
			types.NewMsgRegisterCounterparty(
				"testclientid-1",
				[][]byte{},
				"testclientid-3",
				signer,
			),
			types.ErrInvalidCounterparty,
		},
		{
			"failure: empty signer",
			types.NewMsgRegisterCounterparty(
				"testclientid-2",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-3",
				"badsigner",
			),
			ibcerrors.ErrInvalidAddress,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}
