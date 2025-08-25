package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestPortID defines a reusable port identifier for testing purposes
	TestPortID, _ = types.NewControllerPortID(TestOwnerAddress)
)

type TypesTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (s *TypesTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestTypesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

func (s *TypesTestSuite) TestGenerateAddress() {
	addr := types.GenerateAddress(s.chainA.GetContext(), "test-connection-id", "test-port-id")
	accAddr, err := sdk.AccAddressFromBech32(addr.String())

	s.Require().NoError(err, "TestGenerateAddress failed")
	s.Require().NotEmpty(accAddr)
}

func (s *TypesTestSuite) TestValidateAccountAddress() {
	testCases := []struct {
		name     string
		address  string
		expError error
	}{
		{
			"success",
			TestOwnerAddress,
			nil,
		},
		{
			"success with single character",
			"a",
			nil,
		},
		{
			"empty string",
			"",
			types.ErrInvalidAccountAddress,
		},
		{
			"only spaces",
			"     ",
			types.ErrInvalidAccountAddress,
		},
		{
			"address is too long",
			ibctesting.GenerateString(uint(types.DefaultMaxAddrLength) + 1),
			types.ErrInvalidAccountAddress,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := types.ValidateAccountAddress(tc.address)

			if tc.expError == nil {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().ErrorIs(err, tc.expError, tc.name)
			}
		})
	}
}

func (s *TypesTestSuite) TestInterchainAccount() {
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubkey.Address())
	baseAcc := authtypes.NewBaseAccountWithAddress(addr)
	interchainAcc := types.NewInterchainAccount(baseAcc, TestOwnerAddress)

	// should fail when trying to set the public key or sequence of an interchain account
	err := interchainAcc.SetPubKey(pubkey)
	s.Require().Error(err)
	err = interchainAcc.SetSequence(1)
	s.Require().Error(err)
}

func (s *TypesTestSuite) TestGenesisAccountValidate() {
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubkey.Address())
	baseAcc := authtypes.NewBaseAccountWithAddress(addr)
	pubkey = secp256k1.GenPrivKey().PubKey()
	ownerAddr := sdk.AccAddress(pubkey.Address())

	testCases := []struct {
		name   string
		acc    authtypes.GenesisAccount
		expErr error
	}{
		{
			"success",
			types.NewInterchainAccount(baseAcc, ownerAddr.String()),
			nil,
		},
		{
			"interchain account with empty AccountOwner field",
			types.NewInterchainAccount(baseAcc, ""),
			types.ErrInvalidAccountAddress,
		},
	}

	for _, tc := range testCases {
		err := tc.acc.Validate()

		if tc.expErr == nil {
			s.Require().NoError(err)
		} else {
			s.Require().Error(err)
			s.Require().ErrorIs(err, tc.expErr)
		}
	}
}

func (s *TypesTestSuite) TestInterchainAccountMarshalYAML() {
	addr := s.chainA.SenderAccount.GetAddress()
	baseAcc := authtypes.NewBaseAccountWithAddress(addr)

	interchainAcc := types.NewInterchainAccount(baseAcc, s.chainB.SenderAccount.GetAddress().String())
	bz, err := interchainAcc.MarshalYAML()
	s.Require().NoError(err)

	expected := fmt.Sprintf("address: %s\npublic_key: \"\"\naccount_number: 0\nsequence: 0\naccount_owner: %s\n", s.chainA.SenderAccount.GetAddress(), s.chainB.SenderAccount.GetAddress())
	s.Require().Equal(expected, string(bz))
}

func (s *TypesTestSuite) TestInterchainAccountJSON() {
	addr := s.chainA.SenderAccount.GetAddress()
	ba := authtypes.NewBaseAccountWithAddress(addr)

	interchainAcc := types.NewInterchainAccount(ba, s.chainB.SenderAccount.GetAddress().String())

	bz, err := json.Marshal(interchainAcc)
	s.Require().NoError(err)

	bz1, err := interchainAcc.MarshalJSON()
	s.Require().NoError(err)
	s.Require().Equal(string(bz), string(bz1))

	var a types.InterchainAccount
	s.Require().NoError(json.Unmarshal(bz, &a))
	s.Require().Equal(a.String(), interchainAcc.String())
}
