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

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
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

func NewICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = types.PortID
	path.EndpointB.ChannelConfig.PortID = types.PortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = types.Version
	path.EndpointB.ChannelConfig.Version = types.Version

	return path
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestGeneratePortID() {
	var (
		path  *ibctesting.Path
		owner string
	)
	var testCases = []struct {
		name     string
		malleate func()
		expValue string
		expPass  bool
	}{
		{"success", func() {}, "ics-27-0-0-owner123", true},
		{"success with non matching connection sequences", func() {
			path.EndpointA.ConnectionID = "connection-1"
		}, "ics-27-1-0-owner123", true},
		{"invalid owner address", func() {
			owner = "    "
		}, "", false},
		{"invalid connectionID", func() {
			path.EndpointA.ConnectionID = "connection"
		}, "", false},
		{"invalid counterparty connectionID", func() {
			path.EndpointB.ConnectionID = "connection"
		}, "", false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)
			owner = "owner123" // must be explicitly changed

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
	interchainAcc := types.NewInterchainAccount(baseAcc, "account-owner-id")

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
	suite.SetupTest() // reset

	addr := suite.chainA.SenderAccount.GetAddress()
	ba := authtypes.NewBaseAccountWithAddress(addr)

	interchainAcc := types.NewInterchainAccount(ba, suite.chainB.SenderAccount.GetAddress().String())

	bs, err := yaml.Marshal(interchainAcc)
	suite.Require().NoError(err)

	want := fmt.Sprintf("|\n  address: %s\n  public_key: \"\"\n  account_number: 0\n  sequence: 0\n  account_owner: %s\n", addr, interchainAcc.AccountOwner)
	suite.Require().Equal(want, string(bs))
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
	suite.Require().Equal(interchainAcc.String(), a.String())
}
