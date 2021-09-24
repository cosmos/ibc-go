package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v2/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"
)

type TypesTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *TypesTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestGenerateAddress() {
	addr := types.GenerateAddress("test-port-id")
	accAddr, err := sdk.AccAddressFromBech32(addr.String())

	suite.Require().NoError(err, "TestGenerateAddress failed")
	suite.Require().NotEmpty(accAddr)
}

func (suite *TypesTestSuite) TestGeneratePortID() {
	var (
		path  *ibctesting.Path
		owner = TestOwnerAddress
	)

	testCases := []struct {
		name     string
		malleate func()
		expValue string
		expPass  bool
	}{
		{
			"success",
			func() {},
			fmt.Sprintf("%s-0-0-%s", types.VersionPrefix, TestOwnerAddress),
			true,
		},
		{
			"success with non matching connection sequences",
			func() {
				path.EndpointA.ConnectionID = "connection-1"
			},
			fmt.Sprintf("%s-1-0-%s", types.VersionPrefix, TestOwnerAddress),
			true,
		},
		{
			"invalid owner address",
			func() {
				owner = "    "
			},
			"",
			false,
		},
		{
			"invalid connectionID",
			func() {
				path.EndpointA.ConnectionID = "connection"
			},
			"",
			false,
		},
		{
			"invalid counterparty connectionID",
			func() {
				path.EndpointB.ConnectionID = "connection"
			},
			"",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			tc.malleate()

			portID, err := types.GeneratePortID(owner, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(tc.expValue, portID)
			} else {
				suite.Require().Error(err, tc.name)
				suite.Require().Empty(portID)
			}
		})
	}
}

func (suite *TypesTestSuite) TestInterchainAccount() {
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubkey.Address())
	baseAcc := authtypes.NewBaseAccountWithAddress(addr)
	interchainAcc := types.NewInterchainAccount(baseAcc, TestOwnerAddress)

	// should fail when trying to set the public key or sequence of an interchain account
	err := interchainAcc.SetPubKey(pubkey)
	suite.Require().Error(err)
	err = interchainAcc.SetSequence(1)
	suite.Require().Error(err)
}

func (suite *TypesTestSuite) TestGenesisAccountValidate() {
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubkey.Address())
	baseAcc := authtypes.NewBaseAccountWithAddress(addr)
	pubkey = secp256k1.GenPrivKey().PubKey()
	ownerAddr := sdk.AccAddress(pubkey.Address())

	testCases := []struct {
		name    string
		acc     authtypes.GenesisAccount
		expPass bool
	}{
		{
			"success",
			types.NewInterchainAccount(baseAcc, ownerAddr.String()),
			true,
		},
		{
			"interchain account with empty AccountOwner field",
			types.NewInterchainAccount(baseAcc, ""),
			false,
		},
	}

	for _, tc := range testCases {
		err := tc.acc.Validate()

		if tc.expPass {
			suite.Require().NoError(err)
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *TypesTestSuite) TestInterchainAccountMarshalYAML() {
	addr := suite.chainA.SenderAccount.GetAddress()
	ba := authtypes.NewBaseAccountWithAddress(addr)

	interchainAcc := types.NewInterchainAccount(ba, suite.chainB.SenderAccount.GetAddress().String())
	bz, err := yaml.Marshal(types.InterchainAccountPretty{
		Address:       addr,
		PubKey:        "",
		AccountNumber: interchainAcc.AccountNumber,
		Sequence:      interchainAcc.Sequence,
		AccountOwner:  interchainAcc.AccountOwner,
	})
	suite.Require().NoError(err)

	bz1, err := interchainAcc.MarshalYAML()
	suite.Require().Equal(string(bz), string(bz1))
}

func (suite *TypesTestSuite) TestInterchainAccountJSON() {
	addr := suite.chainA.SenderAccount.GetAddress()
	ba := authtypes.NewBaseAccountWithAddress(addr)

	interchainAcc := types.NewInterchainAccount(ba, suite.chainB.SenderAccount.GetAddress().String())

	bz, err := json.Marshal(interchainAcc)
	suite.Require().NoError(err)

	bz1, err := interchainAcc.MarshalJSON()
	suite.Require().NoError(err)
	suite.Require().Equal(string(bz), string(bz1))

	var a types.InterchainAccount
	suite.Require().NoError(json.Unmarshal(bz, &a))
	suite.Require().Equal(a.String(), interchainAcc.String())
}
