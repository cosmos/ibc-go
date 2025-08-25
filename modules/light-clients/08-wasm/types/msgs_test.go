package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestMsgStoreCodeValidateBasic(t *testing.T) {
	signer := sdk.AccAddress(ibctesting.TestAccAddress).String()
	testCases := []struct {
		name   string
		msg    *types.MsgStoreCode
		expErr error
	}{
		{
			"success: valid signer address, valid length code",
			types.NewMsgStoreCode(signer, wasmtesting.Code),
			nil,
		},
		{
			"failure: code is empty",
			types.NewMsgStoreCode(signer, []byte("")),
			types.ErrWasmEmptyCode,
		},
		{
			"failure: code is too large",
			types.NewMsgStoreCode(signer, make([]byte, types.MaxWasmSize+1)),
			types.ErrWasmCodeTooLarge,
		},
		{
			"failure: signer is invalid",
			types.NewMsgStoreCode("invalid", wasmtesting.Code),
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, tc.expErr)
		}
	}
}

func (s *TypesTestSuite) TestMsgStoreCodeGetSigners() {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		expErr  error
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), nil},
		{"failure: nil address", nil, errors.New("empty address string is not allowed")},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			address := tc.address
			msg := types.NewMsgStoreCode(address.String(), wasmtesting.Code)

			signers, _, err := GetSimApp(s.chainA).AppCodec().GetMsgV1Signers(msg)
			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(address.Bytes(), signers[0])
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func TestMsgMigrateContractValidateBasic(t *testing.T) {
	signer := sdk.AccAddress(ibctesting.TestAccAddress).String()
	validChecksum, err := types.CreateChecksum(wasmtesting.Code)
	require.NoError(t, err, t.Name())
	validMigrateMsg := []byte("{}")

	testCases := []struct {
		name   string
		msg    *types.MsgMigrateContract
		expErr error
	}{
		{
			"success: valid signer address, valid checksum, valid migrate msg",
			types.NewMsgMigrateContract(signer, defaultWasmClientID, validChecksum, validMigrateMsg),
			nil,
		},
		{
			"failure: invalid signer address",
			types.NewMsgMigrateContract(ibctesting.InvalidID, defaultWasmClientID, validChecksum, validMigrateMsg),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: clientID is not a valid client identifier",
			types.NewMsgMigrateContract(signer, ibctesting.InvalidID, validChecksum, validMigrateMsg),
			host.ErrInvalidID,
		},
		{
			"failure: clientID is not a wasm client identifier",
			types.NewMsgMigrateContract(signer, ibctesting.FirstClientID, validChecksum, validMigrateMsg),
			host.ErrInvalidID,
		},
		{
			"failure: checksum is nil",
			types.NewMsgMigrateContract(signer, defaultWasmClientID, nil, validMigrateMsg),
			errorsmod.Wrap(types.ErrInvalidChecksum, "checksum cannot be empty"),
		},
		{
			"failure: checksum is empty",
			types.NewMsgMigrateContract(signer, defaultWasmClientID, []byte{}, validMigrateMsg),
			errorsmod.Wrap(types.ErrInvalidChecksum, "checksum cannot be empty"),
		},
		{
			"failure: checksum is not 32 bytes",
			types.NewMsgMigrateContract(signer, defaultWasmClientID, []byte{1}, validMigrateMsg),
			errorsmod.Wrapf(types.ErrInvalidChecksum, "expected length of 32 bytes, got %d", 1),
		},
		{
			"failure: migrateMsg is nil",
			types.NewMsgMigrateContract(signer, defaultWasmClientID, validChecksum, nil),
			errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "migrate message cannot be empty"),
		},
		{
			"failure: migrateMsg is empty",
			types.NewMsgMigrateContract(signer, defaultWasmClientID, validChecksum, []byte("")),
			errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "migrate message cannot be empty"),
		},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, tc.expErr, tc.name)
		}
	}
}

func (s *TypesTestSuite) TestMsgMigrateContractGetSigners() {
	checksum, err := types.CreateChecksum(wasmtesting.Code)
	s.Require().NoError(err)

	testCases := []struct {
		name    string
		address sdk.AccAddress
		expErr  error
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), nil},
		{"failure: nil address", nil, errors.New("empty address string is not allowed")},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			address := tc.address
			msg := types.NewMsgMigrateContract(address.String(), defaultWasmClientID, checksum, []byte("{}"))

			signers, _, err := GetSimApp(s.chainA).AppCodec().GetMsgV1Signers(msg)
			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(address.Bytes(), signers[0])
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func TestMsgRemoveChecksumValidateBasic(t *testing.T) {
	signer := sdk.AccAddress(ibctesting.TestAccAddress).String()
	checksum, err := types.CreateChecksum(wasmtesting.Code)
	require.NoError(t, err, t.Name())

	testCases := []struct {
		name   string
		msg    *types.MsgRemoveChecksum
		expErr error
	}{
		{
			"success: valid signer address, valid length checksum",
			types.NewMsgRemoveChecksum(signer, checksum),
			nil,
		},
		{
			"failure: checksum is empty",
			types.NewMsgRemoveChecksum(signer, []byte("")),
			types.ErrInvalidChecksum,
		},
		{
			"failure: checksum is nil",
			types.NewMsgRemoveChecksum(signer, nil),
			types.ErrInvalidChecksum,
		},
		{
			"failure: signer is invalid",
			types.NewMsgRemoveChecksum(ibctesting.InvalidID, checksum),
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()

		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expErr, tc.name)
		}
	}
}

func (s *TypesTestSuite) TestMsgRemoveChecksumGetSigners() {
	checksum, err := types.CreateChecksum(wasmtesting.Code)
	s.Require().NoError(err)

	testCases := []struct {
		name     string
		address  sdk.AccAddress
		expError error
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), nil},
		{"failure: nil address", nil, errors.New("empty address string is not allowed")},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			address := tc.address
			msg := types.NewMsgRemoveChecksum(address.String(), checksum)

			signers, _, err := GetSimApp(s.chainA).AppCodec().GetMsgV1Signers(msg)
			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().Equal(address.Bytes(), signers[0])
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expError.Error())
			}
		})
	}
}
