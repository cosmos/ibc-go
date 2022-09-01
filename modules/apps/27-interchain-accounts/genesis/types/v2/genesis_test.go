package v2_test

import (
	"github.com/stretchr/testify/suite"

	controllertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/genesis/types"
	v2 "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/genesis/types/v2"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestPortID defines a reusable port identifier for testing purposes
	TestPortID, _ = icatypes.NewControllerPortID(TestOwnerAddress)
)

type GenesisTypesTestSuite struct {
	suite.Suite
}

func (suite *GenesisTypesTestSuite) TestValidateControllerGenesisState() {
	var genesisState v2.ControllerGenesisState

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"failed to validate active channel - invalid port identifier",
			func() {
				activeChannels := []genesistypes.ActiveChannel{
					{
						PortId:    "invalid|port",
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				genesisState = v2.NewControllerGenesisState(activeChannels, []genesistypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams(), []v2.MiddlewareEnabled{})
			},
			false,
		},
		{
			"failed to validate active channel - invalid channel identifier",
			func() {
				activeChannels := []genesistypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: "invalid|channel",
					},
				}

				genesisState = v2.NewControllerGenesisState(activeChannels, []genesistypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams(), []v2.MiddlewareEnabled{})
			},
			false,
		},
		{
			"failed to validate registered account - invalid port identifier",
			func() {
				activeChannels := []genesistypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []genesistypes.RegisteredInterchainAccount{
					{
						PortId:         "invalid|port",
						AccountAddress: TestOwnerAddress,
					},
				}

				genesisState = v2.NewControllerGenesisState(activeChannels, registeredAccounts, []string{}, controllertypes.DefaultParams(), []v2.MiddlewareEnabled{})
			},
			false,
		},
		{
			"failed to validate registered account - invalid owner address",
			func() {
				activeChannels := []genesistypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []genesistypes.RegisteredInterchainAccount{
					{
						PortId:         TestPortID,
						AccountAddress: "",
					},
				}

				genesisState = v2.NewControllerGenesisState(activeChannels, registeredAccounts, []string{}, controllertypes.DefaultParams(), []v2.MiddlewareEnabled{})
			},
			false,
		},
		{
			"failed to validate controller ports - invalid port identifier",
			func() {
				activeChannels := []genesistypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []genesistypes.RegisteredInterchainAccount{
					{
						PortId:         TestPortID,
						AccountAddress: TestOwnerAddress,
					},
				}

				genesisState = v2.NewControllerGenesisState(activeChannels, registeredAccounts, []string{"invalid|port"}, controllertypes.DefaultParams(), []v2.MiddlewareEnabled{})
			},
			false,
		},
		{
			"failed to validate middleware enabled channel - invalid port identifier",
			func() {
				middlewareEnabledChannels := []v2.MiddlewareEnabled{
					{
						PortId:    "invalid|port",
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				genesisState = v2.NewControllerGenesisState([]genesistypes.ActiveChannel{}, []genesistypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams(), middlewareEnabledChannels)
			},
			false,
		},
		{
			"failed to validate middleware enabled channel - invalid channel identifier",
			func() {
				middlewareEnabledChannels := []v2.MiddlewareEnabled{
					{
						PortId:    TestPortID,
						ChannelId: "invalid|channel",
					},
				}

				genesisState = v2.NewControllerGenesisState([]genesistypes.ActiveChannel{}, []genesistypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams(), middlewareEnabledChannels)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			genesisState = v2.DefaultControllerGenesis()

			tc.malleate() // malleate mutates test data

			err := genesisState.Validate()

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}
