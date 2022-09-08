package types

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
)

// define constants used for testing
const (
	validPort        = "testportid"
	invalidPort      = "(invalidport1)"
	invalidShortPort = "p"
	// 195 characters
	invalidLongPort = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis eros neque, ultricies vel ligula ac, convallis porttitor elit. Maecenas tincidunt turpis elit, vel faucibus nisl pellentesque sodales"

	validChannel        = "testchannel"
	invalidChannel      = "(invalidchannel1)"
	invalidShortChannel = "invalid"
	invalidLongChannel  = "invalidlongchannelinvalidlongchannelinvalidlongchannelinvalidlongchannel"
)

var (
	addr1       = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	addr2       = sdk.AccAddress("testaddr2").String()
	tooLongAddr = "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"
	emptyAddr   string

	coin             = sdk.NewCoin("atom", sdk.NewInt(100))
	ibcCoin          = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", sdk.NewInt(100))
	invalidIBCCoin   = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554", sdk.NewInt(100))
	invalidDenomCoin = sdk.Coin{Denom: "0atom", Amount: sdk.NewInt(100)}
	zeroCoin         = sdk.Coin{Denom: "atoms", Amount: sdk.NewInt(0)}

	timeoutHeight = clienttypes.NewHeight(0, 10)
)

// TestMsgTransferRoute tests Route for MsgTransfer
func TestMsgTransferRoute(t *testing.T) {
	msg := NewMsgTransfer(validPort, validChannel, coin, addr1, addr2, timeoutHeight, 0)

	require.Equal(t, RouterKey, msg.Route())
}

// TestMsgTransferType tests Type for MsgTransfer
func TestMsgTransferType(t *testing.T) {
	msg := NewMsgTransfer(validPort, validChannel, coin, addr1, addr2, timeoutHeight, 0)

	require.Equal(t, "transfer", msg.Type())
}

func TestMsgTransferGetSignBytes(t *testing.T) {
	msg := NewMsgTransfer(validPort, validChannel, coin, addr1, addr2, timeoutHeight, 0)
	expected := fmt.Sprintf(`{"type":"cosmos-sdk/MsgTransfer","value":{"receiver":"%s","sender":"%s","source_channel":"testchannel","source_port":"testportid","timeout_height":{"revision_height":"10"},"token":{"amount":"100","denom":"atom"}}}`, addr2, addr1)
	require.NotPanics(t, func() {
		res := msg.GetSignBytes()
		require.Equal(t, expected, string(res))
	})
}

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgTransferValidation(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *MsgTransfer
		expPass bool
	}{
		{"valid msg with base denom", NewMsgTransfer(validPort, validChannel, coin, addr1, addr2, timeoutHeight, 0), true},
		{"valid msg with trace hash", NewMsgTransfer(validPort, validChannel, ibcCoin, addr1, addr2, timeoutHeight, 0), true},
		{"invalid ibc denom", NewMsgTransfer(validPort, validChannel, invalidIBCCoin, addr1, addr2, timeoutHeight, 0), false},
		{"too short port id", NewMsgTransfer(invalidShortPort, validChannel, coin, addr1, addr2, timeoutHeight, 0), false},
		{"too long port id", NewMsgTransfer(invalidLongPort, validChannel, coin, addr1, addr2, timeoutHeight, 0), false},
		{"port id contains non-alpha", NewMsgTransfer(invalidPort, validChannel, coin, addr1, addr2, timeoutHeight, 0), false},
		{"too short channel id", NewMsgTransfer(validPort, invalidShortChannel, coin, addr1, addr2, timeoutHeight, 0), false},
		{"too long channel id", NewMsgTransfer(validPort, invalidLongChannel, coin, addr1, addr2, timeoutHeight, 0), false},
		{"channel id contains non-alpha", NewMsgTransfer(validPort, invalidChannel, coin, addr1, addr2, timeoutHeight, 0), false},
		{"invalid denom", NewMsgTransfer(validPort, validChannel, invalidDenomCoin, addr1, addr2, timeoutHeight, 0), false},
		{"zero coin", NewMsgTransfer(validPort, validChannel, zeroCoin, addr1, addr2, timeoutHeight, 0), false},
		{"missing sender address", NewMsgTransfer(validPort, validChannel, coin, emptyAddr, addr2, timeoutHeight, 0), false},
		{"missing recipient address", NewMsgTransfer(validPort, validChannel, coin, addr1, "", timeoutHeight, 0), false},
		{"empty coin", NewMsgTransfer(validPort, validChannel, sdk.Coin{}, addr1, addr2, timeoutHeight, 0), false},
		{"too long recipient address", NewMsgTransfer(validPort, validChannel, coin, addr1, tooLongAddr, timeoutHeight, 0), false},
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

// TestMsgTransferGetSigners tests GetSigners for MsgTransfer
func TestMsgTransferGetSigners(t *testing.T) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	msg := NewMsgTransfer(validPort, validChannel, coin, addr.String(), addr2, timeoutHeight, 0)
	res := msg.GetSigners()

	require.Equal(t, []sdk.AccAddress{addr}, res)
}
