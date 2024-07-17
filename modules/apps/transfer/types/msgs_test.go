package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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

	coin              = ibctesting.TestCoin
	coins             = ibctesting.TestCoins
	ibcCoins          = sdk.NewCoins(sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", sdkmath.NewInt(100)))
	invalidIBCCoins   = sdk.NewCoins(sdk.NewCoin("ibc/7F1D3FCF4AE79E1554", sdkmath.NewInt(100)))
	invalidDenomCoins = []sdk.Coin{{Denom: "0atom", Amount: sdkmath.NewInt(100)}}
	zeroCoins         = []sdk.Coin{{Denom: "atoms", Amount: sdkmath.NewInt(0)}}

	timeoutHeight = clienttypes.NewHeight(0, 10)
)

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgTransferValidation(t *testing.T) {
	testCases := []struct {
		name     string
		msg      *types.MsgTransfer
		expError error
	}{
		{"valid msg with base denom", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), nil},
		{"valid msg with unwind", types.NewMsgTransfer("", "", sdk.NewCoins(coin), sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(true)), nil},
		{"valid msg with trace hash", types.NewMsgTransfer(validPort, validChannel, ibcCoins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), nil},
		{"multidenom", types.NewMsgTransfer(validPort, validChannel, coins.Add(ibcCoins...), sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), nil},
		{"memo with forwarding path hops not empty", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "memo", types.NewForwarding(false, validHop)), nil},
		{"memo with forwarding unwind set to true", types.NewMsgTransfer("", "", sdk.NewCoins(coin), sender, receiver, clienttypes.ZeroHeight(), 100, "memo", types.NewForwarding(true)), nil},
		{"invalid ibc denom", types.NewMsgTransfer(validPort, validChannel, invalidIBCCoins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"too short port id", types.NewMsgTransfer(invalidShortPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), host.ErrInvalidID},
		{"too long port id", types.NewMsgTransfer(invalidLongPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), host.ErrInvalidID},
		{"port id contains non-alpha", types.NewMsgTransfer(invalidPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), host.ErrInvalidID},
		{"too short channel id", types.NewMsgTransfer(validPort, invalidShortChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), host.ErrInvalidID},
		{"too long channel id", types.NewMsgTransfer(validPort, invalidLongChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), host.ErrInvalidID},
		{"too long memo", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, ibctesting.GenerateString(types.MaximumMemoLength+1), nil), types.ErrInvalidMemo},
		{"channel id contains non-alpha", types.NewMsgTransfer(validPort, invalidChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), host.ErrInvalidID},
		{"invalid denom", types.NewMsgTransfer(validPort, validChannel, invalidDenomCoins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"zero coins", types.NewMsgTransfer(validPort, validChannel, zeroCoins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"missing sender address", types.NewMsgTransfer(validPort, validChannel, coins, emptyAddr, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidAddress},
		{"missing recipient address", types.NewMsgTransfer(validPort, validChannel, coins, sender, "", clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidAddress},
		{"too long recipient address", types.NewMsgTransfer(validPort, validChannel, coins, sender, ibctesting.GenerateString(types.MaximumReceiverLength+1), clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidAddress},
		{"empty coins", types.NewMsgTransfer(validPort, validChannel, sdk.NewCoins(), sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"multidenom: invalid denom", types.NewMsgTransfer(validPort, validChannel, coins.Add(invalidDenomCoins...), sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"multidenom: invalid ibc denom", types.NewMsgTransfer(validPort, validChannel, coins.Add(invalidIBCCoins...), sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"multidenom: zero coins", types.NewMsgTransfer(validPort, validChannel, zeroCoins, sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"multidenom: too many coins", types.NewMsgTransfer(validPort, validChannel, make([]sdk.Coin, types.MaximumTokensLength+1), sender, receiver, clienttypes.ZeroHeight(), 100, "", nil), ibcerrors.ErrInvalidCoins},
		{"multidenom: both token and tokens are set", &types.MsgTransfer{validPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, "", coins, nil}, ibcerrors.ErrInvalidCoins},
		{"timeout height must be zero if forwarding path hops is not empty", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, timeoutHeight, 100, "memo", types.NewForwarding(false, validHop)), types.ErrInvalidPacketTimeout},
		{"invalid forwarding info port", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(false, types.NewHop(invalidPort, validChannel))), types.ErrInvalidForwarding},
		{"invalid forwarding info channel", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(false, types.NewHop(validPort, invalidChannel))), types.ErrInvalidForwarding},
		{"invalid forwarding info too many hops", types.NewMsgTransfer(validPort, validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(false, generateHops(types.MaximumNumberOfForwardingHops+1)...)), types.ErrInvalidForwarding},
		{"invalid portID when forwarding is set but unwind is not", types.NewMsgTransfer("", validChannel, coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(false, validHop)), host.ErrInvalidID},
		{"invalid channelID when forwarding is set but unwind is not", types.NewMsgTransfer(validPort, "", coins, sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(false, validHop)), host.ErrInvalidID},
		{"unwind specified but source port is not empty", types.NewMsgTransfer(validPort, "", sdk.NewCoins(coin), sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(true)), types.ErrInvalidForwarding},
		{"unwind specified but source channel is not empty", types.NewMsgTransfer("", validChannel, sdk.NewCoins(coin), sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(true)), types.ErrInvalidForwarding},
		{"unwind specified but more than one coin in the message", types.NewMsgTransfer("", "", coins.Add(sdk.NewCoin("atom", ibctesting.TestCoin.Amount)), sender, receiver, clienttypes.ZeroHeight(), 100, "", types.NewForwarding(true)), ibcerrors.ErrInvalidCoins},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()

			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}

// TestMsgTransferGetSigners tests GetSigners for MsgTransfer
func TestMsgTransferGetSigners(t *testing.T) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg := types.NewMsgTransfer(validPort, validChannel, coins, addr.String(), receiver, timeoutHeight, 0, "", nil)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(transfer.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)
	require.NoError(t, err)
	require.Equal(t, addr.Bytes(), signers[0])
}

// TestMsgUpdateParamsValidateBasic tests ValidateBasic for MsgUpdateParams
func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	testCases := []struct {
		name     string
		msg      *types.MsgUpdateParams
		expError error
	}{
		{"success: valid signer and valid params", types.NewMsgUpdateParams(ibctesting.TestAccAddress, types.DefaultParams()), nil},
		{"failure: invalid signer with valid params", types.NewMsgUpdateParams(invalidAddress, types.DefaultParams()), ibcerrors.ErrInvalidAddress},
		{"failure: empty signer with valid params", types.NewMsgUpdateParams(emptyAddr, types.DefaultParams()), ibcerrors.ErrInvalidAddress},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()

			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
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
		t.Run(tc.name, func(t *testing.T) {
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
		})
	}
}
