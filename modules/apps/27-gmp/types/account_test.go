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
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
				ClientId: ibctesting.FirstClientID,
				Sender:   "cosmos1sender",
				Salt:     []byte("randomsalt"),
			},
			nil,
		},
		{
			"success: empty salt is allowed",
			&types.AccountIdentifier{
				ClientId: ibctesting.FirstClientID,
				Sender:   "cosmos1sender",
				Salt:     []byte{},
			},
			nil,
		},
		{
			"success: nil salt is allowed",
			&types.AccountIdentifier{
				ClientId: ibctesting.FirstClientID,
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
				ClientId: ibctesting.FirstClientID,
				Sender:   "",
				Salt:     []byte("salt"),
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: whitespace-only sender",
			&types.AccountIdentifier{
				ClientId: ibctesting.FirstClientID,
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

	t.Run("determinism", func(t *testing.T) {
		accountID := &types.AccountIdentifier{
			ClientId: ibctesting.FirstClientID,
			Sender:   "cosmos1sender",
			Salt:     []byte("randomsalt"),
		}
		firstAddr, err := types.BuildAddressPredictable(accountID)
		require.NoError(t, err)

		for range 50 {
			addr, err := types.BuildAddressPredictable(accountID)
			require.NoError(t, err)
			require.Equal(t, firstAddr, addr)
		}
	})

	t.Run("uniqueness: different salt", func(t *testing.T) {
		addr1, err := types.BuildAddressPredictable(&types.AccountIdentifier{
			ClientId: ibctesting.FirstClientID, Sender: "cosmos1sender", Salt: []byte("salt1"),
		})
		require.NoError(t, err)
		addr2, err := types.BuildAddressPredictable(&types.AccountIdentifier{
			ClientId: ibctesting.FirstClientID, Sender: "cosmos1sender", Salt: []byte("salt2"),
		})
		require.NoError(t, err)
		require.NotEqual(t, addr1, addr2)
	})

	t.Run("uniqueness: different sender", func(t *testing.T) {
		addr1, err := types.BuildAddressPredictable(&types.AccountIdentifier{
			ClientId: ibctesting.FirstClientID, Sender: "cosmos1sender", Salt: []byte("salt"),
		})
		require.NoError(t, err)
		addr2, err := types.BuildAddressPredictable(&types.AccountIdentifier{
			ClientId: ibctesting.FirstClientID, Sender: "cosmos1different", Salt: []byte("salt"),
		})
		require.NoError(t, err)
		require.NotEqual(t, addr1, addr2)
	})

	t.Run("uniqueness: different client ID", func(t *testing.T) {
		addr1, err := types.BuildAddressPredictable(&types.AccountIdentifier{
			ClientId: ibctesting.FirstClientID, Sender: "cosmos1sender", Salt: []byte("salt"),
		})
		require.NoError(t, err)
		addr2, err := types.BuildAddressPredictable(&types.AccountIdentifier{
			ClientId: "07-tendermint-1", Sender: "cosmos1sender", Salt: []byte("salt"),
		})
		require.NoError(t, err)
		require.NotEqual(t, addr1, addr2)
	})
}

func TestNewAccountIdentifier(t *testing.T) {
	clientID := ibctesting.FirstClientID
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
		ClientId: ibctesting.FirstClientID,
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

	t.Run("failure: invalid codec", func(t *testing.T) {
		mock := &mockCodec{}
		msgs, err := types.DeserializeCosmosTx(mock, []byte("data"))
		require.ErrorIs(t, err, types.ErrInvalidCodec)
		require.Nil(t, msgs)
	})
}

// mockCodec is a mock codec that implements codec.Codec but is not a ProtoCodec
type mockCodec struct {
	codec.Codec
}
