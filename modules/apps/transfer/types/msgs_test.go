package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
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

	invalidAddress = "invalid"
)

var (
	sender    = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver  = sdk.AccAddress("testaddr2").String()
	emptyAddr string

	coins             = sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(100)))
	ibcCoins          = sdk.NewCoins(sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", sdkmath.NewInt(100)))
	invalidIBCCoins   = sdk.NewCoins(sdk.NewCoin("ibc/7F1D3FCF4AE79E1554", sdkmath.NewInt(100)))
	invalidDenomCoins = []sdk.Coin{{Denom: "0atom", Amount: sdkmath.NewInt(100)}}
	zeroCoins         = sdk.NewCoins(sdk.Coin{Denom: "atoms", Amount: sdkmath.NewInt(0)})
	timeoutHeight     = clienttypes.NewHeight(0, 10)
)

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgTransferValidation(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *types.MsgTransfer
		expPass bool
	}{
		{"valid msg with base denom", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), true},
		{"valid msg with trace hash", types.NewMsgTransfer(validPort, validChannel, ibcCoins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), true},
		{"multidenom", types.NewMsgTransfer(validPort, validChannel, coins.Add(ibcCoins...), sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), true},
		{"invalid ibc denom", types.NewMsgTransfer(validPort, validChannel, invalidIBCCoins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"too short port id", types.NewMsgTransfer(invalidShortPort, validChannel, coins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"too long port id", types.NewMsgTransfer(invalidLongPort, validChannel, coins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"port id contains non-alpha", types.NewMsgTransfer(invalidPort, validChannel, coins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"too short channel id", types.NewMsgTransfer(validPort, invalidShortChannel, coins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"too long channel id", types.NewMsgTransfer(validPort, invalidLongChannel, coins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"too long memo", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, timeoutHeight, 0, ibctesting.GenerateString(types.MaximumMemoLength+1), ibctesting.TestEmptyForwardingPath), false},
		{"channel id contains non-alpha", types.NewMsgTransfer(validPort, invalidChannel, coins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"invalid denom", types.NewMsgTransfer(validPort, validChannel, invalidDenomCoins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"zero coins", types.NewMsgTransfer(validPort, validChannel, zeroCoins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"missing sender address", types.NewMsgTransfer(validPort, validChannel, coins, emptyAddr, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"missing recipient address", types.NewMsgTransfer(validPort, validChannel, coins, sender, "", timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"too long recipient address", types.NewMsgTransfer(validPort, validChannel, coins, sender, ibctesting.GenerateString(types.MaximumReceiverLength+1), timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"empty coins", types.NewMsgTransfer(validPort, validChannel, sdk.NewCoins(), sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"multidenom: invalid denom", types.NewMsgTransfer(validPort, validChannel, coins.Add(invalidDenomCoins...), sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"multidenom: invalid ibc denom", types.NewMsgTransfer(validPort, validChannel, coins.Add(invalidIBCCoins...), sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
		{"multidenom: zero coins", types.NewMsgTransfer(validPort, validChannel, zeroCoins, sender, receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath), false},
	}

	for i, tc := range testCases {
		tc := tc

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
	msg := types.NewMsgTransfer(validPort, validChannel, coins, addr.String(), receiver, timeoutHeight, 0, "", ibctesting.TestEmptyForwardingPath)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(transfer.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)
	require.NoError(t, err)
	require.Equal(t, addr.Bytes(), signers[0])
}

// TestMsgUpdateParamsValidateBasic tests ValidateBasic for MsgUpdateParams
func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{"success: valid signer and valid params", types.NewMsgUpdateParams(ibctesting.TestAccAddress, types.DefaultParams()), true},
		{"failure: invalid signer with valid params", types.NewMsgUpdateParams(invalidAddress, types.DefaultParams()), false},
		{"failure: empty signer with valid params", types.NewMsgUpdateParams(emptyAddr, types.DefaultParams()), false},
	}

	for i, tc := range testCases {
		tc := tc

		err := tc.msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// TestMsgUpdateParamsGetSigners tests GetSigners for MsgUpdateParams
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
		tc := tc

		msg := types.MsgUpdateParams{
			Signer: tc.address.String(),
			Params: types.DefaultParams(),
		}

		encodingCfg := moduletestutil.MakeTestEncodingConfig(transfer.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(&msg)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.Error(t, err)
		}
	}
}
