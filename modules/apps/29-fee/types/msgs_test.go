package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

var (
	validAadr1   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	validAadr2   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	invalidAadr1 = "invalid_address"
	invalidAadr2 = "invalid_address"
)

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgRegisterCountepartyAddressValidation(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *MsgRegisterCounterpartyAddress
		expPass bool
	}{
		{"validate with correct sdk.AccAddress", NewMsgRegisterCounterpartyAddress(validAadr1, validAadr2), true},
		{"validate with incorrect source relayer address", NewMsgRegisterCounterpartyAddress(invalidAadr1, validAadr2), false},
		{"validate with incorrect counterparty source relayer address", NewMsgRegisterCounterpartyAddress(validAadr1, invalidAadr2), false},
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
