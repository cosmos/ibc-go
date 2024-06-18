package types_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
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
	sender    = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver  = sdk.AccAddress("testreceiver").String()
	emptyAddr string

	coin             = sdk.NewCoin("atom", sdk.NewInt(100))
	ibcCoin          = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", sdk.NewInt(100))
	invalidIBCCoin   = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554", sdk.NewInt(100))
	invalidDenomCoin = sdk.Coin{Denom: "0atom", Amount: sdk.NewInt(100)}
	zeroCoin         = sdk.Coin{Denom: "atoms", Amount: sdk.NewInt(0)}

	timeoutHeight = clienttypes.NewHeight(0, 10)
)

// TestMsgTransferRoute tests Route for MsgTransfer
func TestMsgTransferRoute(t *testing.T) {
	msg := types.NewMsgTransfer(validPort, validChannel, coin, sender, receiver, timeoutHeight, 0, "")

	require.Equal(t, types.RouterKey, msg.Route())
}

func TestMsgTransferGetSignBytes(t *testing.T) {
	msg := types.NewMsgTransfer(validPort, validChannel, coin, sender, receiver, timeoutHeight, 0, "")
	expected := fmt.Sprintf(`{"type":"cosmos-sdk/MsgTransfer","value":{"receiver":"%s","sender":"%s","source_channel":"testchannel","source_port":"testportid","timeout_height":{"revision_height":"10"},"token":{"amount":"100","denom":"atom"}}}`, receiver, sender)
	require.NotPanics(t, func() {
		res := msg.GetSignBytes()
		require.Equal(t, expected, string(res))
	})
}

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgTransferValidation(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *types.MsgTransfer
		expPass bool
	}{
		{"valid msg with base denom", types.NewMsgTransfer(validPort, validChannel, coin, sender, receiver, timeoutHeight, 0, ""), true},
		{"valid msg with trace hash", types.NewMsgTransfer(validPort, validChannel, ibcCoin, sender, receiver, timeoutHeight, 0, ""), true},
		{"invalid ibc denom", types.NewMsgTransfer(validPort, validChannel, invalidIBCCoin, sender, receiver, timeoutHeight, 0, ""), false},
		{"too short port id", types.NewMsgTransfer(invalidShortPort, validChannel, coin, sender, receiver, timeoutHeight, 0, ""), false},
		{"too long port id", types.NewMsgTransfer(invalidLongPort, validChannel, coin, sender, receiver, timeoutHeight, 0, ""), false},
		{"port id contains non-alpha", types.NewMsgTransfer(invalidPort, validChannel, coin, sender, receiver, timeoutHeight, 0, ""), false},
		{"too short channel id", types.NewMsgTransfer(validPort, invalidShortChannel, coin, sender, receiver, timeoutHeight, 0, ""), false},
		{"too long channel id", types.NewMsgTransfer(validPort, invalidLongChannel, coin, sender, receiver, timeoutHeight, 0, ""), false},
		{"too long memo", types.NewMsgTransfer(validPort, validChannel, coin, sender, receiver, timeoutHeight, 0, ibctesting.GenerateString(types.MaximumMemoLength+1)), false},
		{"channel id contains non-alpha", types.NewMsgTransfer(validPort, invalidChannel, coin, sender, receiver, timeoutHeight, 0, ""), false},
		{"invalid denom", types.NewMsgTransfer(validPort, validChannel, invalidDenomCoin, sender, receiver, timeoutHeight, 0, ""), false},
		{"zero coin", types.NewMsgTransfer(validPort, validChannel, zeroCoin, sender, receiver, timeoutHeight, 0, ""), false},
		{"missing sender address", types.NewMsgTransfer(validPort, validChannel, coin, emptyAddr, receiver, timeoutHeight, 0, ""), false},
		{"missing recipient address", types.NewMsgTransfer(validPort, validChannel, coin, sender, "", timeoutHeight, 0, ""), false},
		{"too long recipient address", types.NewMsgTransfer(validPort, validChannel, coin, sender, ibctesting.GenerateString(types.MaximumReceiverLength+1), timeoutHeight, 0, ""), false},
		{"empty coin", types.NewMsgTransfer(validPort, validChannel, sdk.Coin{}, sender, receiver, timeoutHeight, 0, ""), false},
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

	msg := types.NewMsgTransfer(validPort, validChannel, coin, addr.String(), receiver, timeoutHeight, 0, "")
	res := msg.GetSigners()

	require.Equal(t, []sdk.AccAddress{addr}, res)
}
