package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// define constants used for testing
const (
	validPort        = "testportid"
	invalidPort      = "(invalidport1)"
	invalidShortPort = "p"
	// 195 characters
	invalidLongPort = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis eros neque, ultricies vel ligula ac, convallis porttitor elit. Maecenas tincidunt turpis elit, vel faucibus nisl pellentesque sodales"

	validChannel        = "channel-5"
	eurekaClient        = "07-tendermint-0"
	invalidChannel      = "(invalidchannel1)"
	invalidShortChannel = "invalid"
	invalidLongChannel  = "invalidlongchannelinvalidlongchannelinvalidlongchannelinvalidlongchannel"

	invalidAddress = "invalid"
)

var (
	sender    = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver  = sdk.AccAddress("testaddr2").String()
	emptyAddr string

	coin             = ibctesting.TestCoin
	ibcCoin          = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", sdkmath.NewInt(100))
	invalidIBCCoin   = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554", sdkmath.NewInt(100))
	invalidDenomCoin = sdk.Coin{Denom: "0atom", Amount: sdkmath.NewInt(100)}
	zeroCoin         = sdk.Coin{Denom: "atoms", Amount: sdkmath.NewInt(0)}

	timeoutHeight = clienttypes.NewHeight(0, 10)
)

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgTransferValidation(t *testing.T) {
	testCases := []struct {
		name     string
		msg      *types.MsgTransfer
		expError error
	}{
		{"valid msg with base denom", types.NewMsgTransfer(validPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), nil},
		{"valid aliased channel", types.NewMsgTransferAliased(validPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), nil},
		{"valid aliased channel with encoding", types.NewMsgTransferWithEncoding(validPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, "", "application/json", true), nil},
		{"valid eureka msg with base denom", types.NewMsgTransfer(validPort, eurekaClient, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), nil},
		{"valid eureka msg with base denom and encoding", types.NewMsgTransferWithEncoding(validPort, eurekaClient, coin, sender, receiver, clienttypes.ZeroHeight(), 100, "", "application/json", false), nil},
		{"valid msg with trace hash", types.NewMsgTransfer(validPort, validChannel, ibcCoin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), nil},
		{"valid eureka msg with trace hash", types.NewMsgTransfer(validPort, eurekaClient, ibcCoin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), nil},
		{"valid eureka msg with trace hash with encoding", types.NewMsgTransferWithEncoding(validPort, eurekaClient, ibcCoin, sender, receiver, clienttypes.ZeroHeight(), 100, "", "application/json", false), nil},
		{"invalid ibc denom", types.NewMsgTransfer(validPort, validChannel, invalidIBCCoin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), ibcerrors.ErrInvalidCoins},
		{"too short port id", types.NewMsgTransfer(invalidShortPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), host.ErrInvalidID},
		{"too long port id", types.NewMsgTransfer(invalidLongPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), host.ErrInvalidID},
		{"port id contains non-alpha", types.NewMsgTransfer(invalidPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), host.ErrInvalidID},
		{"too short channel id", types.NewMsgTransfer(validPort, invalidShortChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), host.ErrInvalidID},
		{"too long channel id", types.NewMsgTransfer(validPort, invalidLongChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), host.ErrInvalidID},
		{"too long memo", types.NewMsgTransfer(validPort, validChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ibctesting.GenerateString(types.MaximumMemoLength+1)), types.ErrInvalidMemo},
		{"channel id contains non-alpha", types.NewMsgTransfer(validPort, invalidChannel, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), host.ErrInvalidID},
		{"invalid denom", types.NewMsgTransfer(validPort, validChannel, invalidDenomCoin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), ibcerrors.ErrInvalidCoins},
		{"zero coin", types.NewMsgTransfer(validPort, validChannel, zeroCoin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), ibcerrors.ErrInvalidCoins},
		{"missing sender address", types.NewMsgTransfer(validPort, validChannel, coin, emptyAddr, receiver, clienttypes.ZeroHeight(), 100, ""), ibcerrors.ErrInvalidAddress},
		{"missing recipient address", types.NewMsgTransfer(validPort, validChannel, coin, sender, "", clienttypes.ZeroHeight(), 100, ""), ibcerrors.ErrInvalidAddress},
		{"too long recipient address", types.NewMsgTransfer(validPort, validChannel, coin, sender, ibctesting.GenerateString(types.MaximumReceiverLength+1), clienttypes.ZeroHeight(), 100, ""), ibcerrors.ErrInvalidAddress},
		{"empty coin", types.NewMsgTransfer(validPort, validChannel, sdk.Coin{}, sender, receiver, clienttypes.ZeroHeight(), 100, ""), ibcerrors.ErrInvalidCoins},
		{"invalid aliased channel", types.NewMsgTransferAliased(validPort, eurekaClient, coin, sender, receiver, clienttypes.ZeroHeight(), 100, ""), host.ErrInvalidID},
	}

	for _, tc := range testCases {
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

// TestMsgTransferGetSigners tests GetSigners for MsgTransfer
func TestMsgTransferGetSigners(t *testing.T) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg := types.NewMsgTransfer(validPort, validChannel, coin, addr.String(), receiver, timeoutHeight, 0, "")

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

			if tc.expError == nil {
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
		errMsg  string
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), ""},
		{"failure: nil address", nil, "empty address string is not allowed"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := types.MsgUpdateParams{
				Signer: tc.address.String(),
				Params: types.DefaultParams(),
			}

			encodingCfg := moduletestutil.MakeTestEncodingConfig(transfer.AppModuleBasic{})
			signers, _, err := encodingCfg.Codec.GetMsgV1Signers(&msg)

			if tc.errMsg == "" {
				require.NoError(t, err)
				require.Equal(t, tc.address.Bytes(), signers[0])
			} else {
				require.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}
