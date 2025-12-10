package types_test

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	gmp "github.com/cosmos/ibc-go/v10/modules/apps/27-gmp"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

func TestBuildAddressPredictable(t *testing.T) {
	testCases := []struct {
		name      string
		accountID *types.AccountIdentifier
		expErr    error
	}{
		{
			"success: valid account identifier",
			&types.AccountIdentifier{
				ClientId: "07-tendermint-0",
				Sender:   "cosmos1sender",
				Salt:     []byte("randomsalt"),
			},
			nil,
		},
		{
			"success: empty salt is allowed",
			&types.AccountIdentifier{
				ClientId: "07-tendermint-0",
				Sender:   "cosmos1sender",
				Salt:     []byte{},
			},
			nil,
		},
		{
			"success: nil salt is allowed",
			&types.AccountIdentifier{
				ClientId: "07-tendermint-0",
				Sender:   "cosmos1sender",
				Salt:     nil,
			},
			nil,
		},
		{
			"failure: invalid client ID format - too short",
			&types.AccountIdentifier{
				ClientId: "abc",
				Sender:   "cosmos1sender",
				Salt:     []byte("salt"),
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty client ID",
			&types.AccountIdentifier{
				ClientId: "",
				Sender:   "cosmos1sender",
				Salt:     []byte("salt"),
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty sender",
			&types.AccountIdentifier{
				ClientId: "07-tendermint-0",
				Sender:   "",
				Salt:     []byte("salt"),
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: whitespace-only sender",
			&types.AccountIdentifier{
				ClientId: "07-tendermint-0",
				Sender:   "   ",
				Salt:     []byte("salt"),
			},
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := types.BuildAddressPredictable(tc.accountID)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				require.Nil(t, addr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, addr)
				require.Len(t, addr, types.AccountAddrLen, "address should be exactly %d bytes", types.AccountAddrLen)
			}
		})
	}
}

func TestBuildAddressPredictable_Determinism(t *testing.T) {
	accountID := &types.AccountIdentifier{
		ClientId: "07-tendermint-0",
		Sender:   "cosmos1sender",
		Salt:     []byte("randomsalt"),
	}

	// Generate address multiple times
	addr1, err := types.BuildAddressPredictable(accountID)
	require.NoError(t, err)

	addr2, err := types.BuildAddressPredictable(accountID)
	require.NoError(t, err)

	addr3, err := types.BuildAddressPredictable(accountID)
	require.NoError(t, err)

	// All addresses should be identical
	require.Equal(t, addr1, addr2, "addresses should be deterministic")
	require.Equal(t, addr2, addr3, "addresses should be deterministic")
}

func TestBuildAddressPredictable_Uniqueness(t *testing.T) {
	// Base account identifier
	baseAccountID := &types.AccountIdentifier{
		ClientId: "07-tendermint-0",
		Sender:   "cosmos1sender",
		Salt:     []byte("salt1"),
	}

	baseAddr, err := types.BuildAddressPredictable(baseAccountID)
	require.NoError(t, err)

	// Different salt should produce different address
	differentSaltID := &types.AccountIdentifier{
		ClientId: "07-tendermint-0",
		Sender:   "cosmos1sender",
		Salt:     []byte("salt2"),
	}
	differentSaltAddr, err := types.BuildAddressPredictable(differentSaltID)
	require.NoError(t, err)
	require.NotEqual(t, baseAddr, differentSaltAddr, "different salts should produce different addresses")

	// Different sender should produce different address
	differentSenderID := &types.AccountIdentifier{
		ClientId: "07-tendermint-0",
		Sender:   "cosmos1different",
		Salt:     []byte("salt1"),
	}
	differentSenderAddr, err := types.BuildAddressPredictable(differentSenderID)
	require.NoError(t, err)
	require.NotEqual(t, baseAddr, differentSenderAddr, "different senders should produce different addresses")

	// Different client ID should produce different address
	differentClientID := &types.AccountIdentifier{
		ClientId: "07-tendermint-1",
		Sender:   "cosmos1sender",
		Salt:     []byte("salt1"),
	}
	differentClientAddr, err := types.BuildAddressPredictable(differentClientID)
	require.NoError(t, err)
	require.NotEqual(t, baseAddr, differentClientAddr, "different client IDs should produce different addresses")
}

func TestNewAccountIdentifier(t *testing.T) {
	clientID := "07-tendermint-0"
	sender := "cosmos1sender"
	salt := []byte("salt")

	accountID := types.NewAccountIdentifier(clientID, sender, salt)

	require.Equal(t, clientID, accountID.ClientId)
	require.Equal(t, sender, accountID.Sender)
	require.Equal(t, salt, accountID.Salt)
}

func TestNewICS27Account(t *testing.T) {
	addr := "cosmos1address"
	accountID := &types.AccountIdentifier{
		ClientId: "07-tendermint-0",
		Sender:   "cosmos1sender",
		Salt:     []byte("salt"),
	}

	account := types.NewICS27Account(addr, accountID)

	require.Equal(t, addr, account.Address)
	require.Equal(t, accountID, account.AccountId)
}

func TestSerializeCosmosTx(t *testing.T) {
	encodingCfg := moduletestutil.MakeTestEncodingConfig(gmp.AppModule{}, bank.AppModule{})
	cdc := encodingCfg.Codec

	msg := &banktypes.MsgSend{
		FromAddress: "cosmos1sender",
		ToAddress:   "cosmos1recipient",
		Amount:      sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(100))),
	}

	testCases := []struct {
		name   string
		msgs   []proto.Message
		expErr bool
	}{
		{
			"success: single message",
			[]proto.Message{msg},
			false,
		},
		{
			"success: multiple messages",
			[]proto.Message{msg, msg},
			false,
		},
		{
			"success: empty messages",
			[]proto.Message{},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bz, err := types.SerializeCosmosTx(cdc, tc.msgs)
			if tc.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, bz)
			}
		})
	}
}

func TestDeserializeCosmosTx(t *testing.T) {
	encodingCfg := moduletestutil.MakeTestEncodingConfig(gmp.AppModule{}, bank.AppModule{})
	cdc := encodingCfg.Codec

	msg := &banktypes.MsgSend{
		FromAddress: "cosmos1sender",
		ToAddress:   "cosmos1recipient",
		Amount:      sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(100))),
	}

	validPayload, err := types.SerializeCosmosTx(cdc, []proto.Message{msg})
	require.NoError(t, err)

	t.Run("success: valid payload", func(t *testing.T) {
		msgs, err := types.DeserializeCosmosTx(cdc, validPayload)
		require.NoError(t, err)
		require.NotNil(t, msgs)
		require.Len(t, msgs, 1)
	})

	t.Run("failure: invalid data", func(t *testing.T) {
		msgs, err := types.DeserializeCosmosTx(cdc, []byte("invalid data"))
		require.ErrorIs(t, err, ibcerrors.ErrInvalidType)
		require.Nil(t, msgs)
	})
}

// mockCodec is a mock codec that implements codec.Codec but is not a ProtoCodec
type mockCodec struct {
	codec.Codec
}

func TestDeserializeCosmosTx_InvalidCodec(t *testing.T) {
	// Test that the function rejects non-ProtoCodec codecs
	mock := &mockCodec{}
	msgs, err := types.DeserializeCosmosTx(mock, []byte("data"))
	require.ErrorIs(t, err, types.ErrInvalidCodec)
	require.Nil(t, msgs)
}
