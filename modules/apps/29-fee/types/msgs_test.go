package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

var (
	validAddr   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	invalidAddr = "invalid_address"
)

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgRegisterCountepartyAddressValidation(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *MsgRegisterCounterpartyAddress
		expPass bool
	}{
		{"validate with correct sdk.AccAddress", NewMsgRegisterCounterpartyAddress(validAddr, validAddr), true},
		{"validate with incorrect destination relayer address", NewMsgRegisterCounterpartyAddress(invalidAddr, validAddr), false},
		{"validate with incorrect counterparty relayer address", NewMsgRegisterCounterpartyAddress(validAddr, invalidAddr), false},
	}

	for i, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
