package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/suite"

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
	addr := types.GenerateAddress([]byte{}, "test-port-id")
	accAddr, err := sdk.AccAddressFromBech32(addr.String())

	suite.Require().NoError(err, "TestGenerateAddress failed")
	suite.Require().NotEmpty(accAddr)
}

func (suite *TypesTestSuite) TestParseAddressFromVersion() {
	version := types.NewAppVersion(types.VersionPrefix, TestOwnerAddress)

	addr := types.ParseAddressFromVersion(version)
	suite.Require().Equal(TestOwnerAddress, addr)

	addr = types.ParseAddressFromVersion("test-version-string")
	suite.Require().Empty(addr)
}

func (suite *TypesTestSuite) TestParseCtrlConnSequence() {
	portID, err := types.GeneratePortID(TestOwnerAddress, "connection-0", "connection-1")
	suite.Require().NoError(err)

	connSeq := types.ParseCtrlConnSequence(portID)
	suite.Require().Equal("0", connSeq)
	suite.Require().Empty(types.ParseCtrlConnSequence(types.PortID))
}

func (suite *TypesTestSuite) TestParseHostConnSequence() {
	portID, err := types.GeneratePortID(TestOwnerAddress, "connection-0", "connection-1")
	suite.Require().NoError(err)

	connSeq := types.ParseHostConnSequence(portID)
	suite.Require().Equal("1", connSeq)
	suite.Require().Empty(types.ParseHostConnSequence(types.PortID))
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
			fmt.Sprintf("%s|0|0|%s", types.VersionPrefix, TestOwnerAddress),
			true,
		},
		{
			"success with non matching connection sequences",
			func() {
				path.EndpointA.ConnectionID = "connection-1"
			},
			fmt.Sprintf("%s|1|0|%s", types.VersionPrefix, TestOwnerAddress),
			true,
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
		{
			"invalid owner address",
			func() {
				owner = "    "
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
	baseAcc := authtypes.NewBaseAccountWithAddress(addr)

	interchainAcc := types.NewInterchainAccount(baseAcc, suite.chainB.SenderAccount.GetAddress().String())
	bz, err := interchainAcc.MarshalYAML()
	suite.Require().NoError(err)

	expected := fmt.Sprintf("address: %s\npublic_key: \"\"\naccount_number: 0\nsequence: 0\naccount_owner: %s\n", suite.chainA.SenderAccount.GetAddress(), suite.chainB.SenderAccount.GetAddress())
	suite.Require().Equal(expected, string(bz))
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
