package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

	coin             = sdk.NewCoin("atom", sdkmath.NewInt(100))
	ibcCoin          = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", sdkmath.NewInt(100))
	invalidIBCCoin   = sdk.NewCoin("ibc/7F1D3FCF4AE79E1554", sdkmath.NewInt(100))
	invalidDenomCoin = sdk.Coin{Denom: "0atom", Amount: sdkmath.NewInt(100)}
	zeroCoin         = sdk.Coin{Denom: "atoms", Amount: sdkmath.NewInt(0)}

	timeoutHeight = clienttypes.NewHeight(0, 10)
)

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
		{"channel id contains non-alpha", types.NewMsgTransfer(validPort, invalidChannel, coin, sender, receiver, timeoutHeight, 0, ""), false},
		{"invalid denom", types.NewMsgTransfer(validPort, validChannel, invalidDenomCoin, sender, receiver, timeoutHeight, 0, ""), false},
		{"zero coin", types.NewMsgTransfer(validPort, validChannel, zeroCoin, sender, receiver, timeoutHeight, 0, ""), false},
		{"missing sender address", types.NewMsgTransfer(validPort, validChannel, coin, emptyAddr, receiver, timeoutHeight, 0, ""), false},
		{"missing recipient address", types.NewMsgTransfer(validPort, validChannel, coin, sender, "", timeoutHeight, 0, ""), false},
		{"too long recipient address", types.NewMsgTransfer(validPort, validChannel, coin, sender, "7146fd910a6340deec3e02ab18071d6501f6f4c824138064a39e6406861f32aa4d6cb9837890e266526bba749d7bbdaecb2d8074d45f23271cf42a5169b259b3f40d16a60b792fb0c44631321bb42f6a8ed4c7ac948c556f0dcfad771c6d2c7dfd0281afaf583edd6e6d0aca06170907a429033fa2c20c95b655659af2f1defdab8a9d5bff3420e560ddedb235282370ce7f1211859d3a42f9428c65d38d6d1dfb111b81e777e731988e1f6326dc48ca05c2afcfa1578f123d4b88f3fa8b19980e2a3a844205d4a1a8784f21c614a19cf9bdd21f477681dfb42e6417ad53a8a48a07e77fd2116cc78e7c6f817ca8676ed039806816a74c5648e6b2261f415deac4947b56c23062eb06a3ccf9de95083e7b52cbe058b2e9ddb42f983df24bd4750ed185fb49235a7ac965624870a90c0fe081b46f1b077d6b27a2be2a74c41c1505194e583e15e3174d33b892571b40f8cffa02831fdce8c2fa0429fcfdb3c4d691f7bc4049e12d2fdc63af0d4bf504e6135f361b5dccc235b1da862292c77d1f9232970b71d10208644a971787df4bbb98618e60c008d5ef86a68994277ac6f09937d85b62b6b95d49af1832d94f0cb74ee2f29bce9b49078da897db747dcb981bba4e074f2da7be4eee0d452c72d8bf33c43ca5b35f6cdd382a8c8852c2d308d3a6cfb7da6ad09515b18b734b942f3c8c73668f12bc3385e8651b1e2e9c474d2f2aa9e9e05f6ec66a9dffb16e62c03b94c68ca692500d5200512d1e6806abfca18dbf1812de659409b8f08ec9188c44423966ad8652a84deb8f875390ea18682f41694a630cfc277be93d13ddafb35d4bb2e31193858f5f0c03525788182300cdd910cd97afc2072735d2bddc73f0a68b7416a345e85291535c239fa8a1064c2a610b822fe1b9372a3e73581644274d7bb739f3005485348302fd7582978e5c9bdb5eec9fd34ba3e951cb1a54cdb5d4ac84b7d489d449143fc3caa7406e143759d3359f6614d03387386a17e6798d9022d8f394b50b03144954504b6561659adb176aa92d6a3eb90d442e45157d41dac5eec8828fcedc007abbe4e635b1fc6093f7baf83ced16599044b9b017511dc0a8815f25e68afdb0cc8788027616e51167ec2186e0098497b662ce16540cb4110a3d29fa07d81e7771c4ce1ef8fec018d622aee2be98e1a7331df43c1e88eff7cf4315ee763c081d9c2ca5e2d13df6a7271113f2ebb6b8153b1fe3c95ea2bc11a5d06b5a24a44747e71bb52cbabb213edb9f9cc3e8389cd642fd3c1feedbf81877142ac2f19ae5d76cb9d8b0129af1171f284ac86f78061544468229b6747231c331ebdd4c5e2b5e78edd5a5f1844a7d492f05e5b5d4b408eb48e72f16d32390e3a499ed6257d098f0a687cabbf4535052a7472271adac900175ce3b9ea3bea07c8341144a7b5f23ed8e868f56f8edce884b0a350", timeoutHeight, 0, ""), false},
		{"empty coin", types.NewMsgTransfer(validPort, validChannel, sdk.Coin{}, sender, receiver, timeoutHeight, 0, ""), false},
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

	msg := types.NewMsgTransfer(validPort, validChannel, coin, addr.String(), receiver, timeoutHeight, 0, "")
	res := msg.GetSigners()

	require.Equal(t, []sdk.AccAddress{addr}, res)
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
		if tc.expPass {
			require.Equal(t, []sdk.AccAddress{tc.address}, msg.GetSigners())
		} else {
			require.Panics(t, func() {
				msg.GetSigners()
			})
		}
	}
}
