package types_test

import (
	"testing"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/stretchr/testify/require"
)

// TestValidateBasicAcknowledgement tests ValidateBasic function of Acknowledgement
func TestValidateBasicAcknowledgment(t *testing.T) {
	var ack types.Acknowledgement
	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				ack = types.Acknowledgement{
					RecvSuccess: true,
					AppAcknowledgements: [][]byte{
						[]byte("some bytes"),
						[]byte("some bytes 2"),
					},
				}
			},
			nil,
		},
		{
			"success with failed receive",
			func() {
				ack = types.Acknowledgement{
					RecvSuccess: false,
					AppAcknowledgements: [][]byte{
						[]byte("some bytes"),
						[]byte("some bytes 2"),
					},
				}
			},
			nil,
		},
		{
			"failure: empty ack",
			func() {
				ack = types.Acknowledgement{}
			},
			types.ErrInvalidAcknowledgement,
		},
		{
			"failure: empty success ack",
			func() {
				ack = types.Acknowledgement{RecvSuccess: true}
			},
			types.ErrInvalidAcknowledgement,
		},
		{
			"failure: empty app ack",
			func() {
				ack = types.Acknowledgement{
					RecvSuccess: false,
					AppAcknowledgements: [][]byte{
						[]byte(""), // empty app ack
					},
				}
			},
			types.ErrInvalidAcknowledgement,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.malleate()

			err := ack.ValidateBasic()
			if tc.expErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expErr)
			}
		})
	}
}
