package types_test

import (
	controllertypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *TypesTestSuite) TestValidateGenesisState() {
	var (
		genesisState types.GenesisState
	)

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
			"failed to validate - empty value",
			func() {
				genesisState = types.GenesisState{}
			},
			false,
		},
		{
			"failed to validate - invalid controller genesis",
			func() {
				genesisState = *types.NewGenesisState(controllertypes.ControllerGenesisState{Ports: []string{"invalid|port"}}, types.DefaultHostGenesis())
			},
			false,
		},
		{
			"failed to validate - invalid host genesis",
			func() {
				genesisState = *types.NewGenesisState(types.DefaultControllerGenesis(), hosttypes.HostGenesisState{})
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			genesisState = *types.DefaultGenesis()

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

func (suite *TypesTestSuite) TestValidateControllerGenesisState() {
	var (
		genesisState controllertypes.ControllerGenesisState
	)

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
				activeChannels := []controllertypes.ActiveChannel{
					{
						PortId:    "invalid|port",
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				genesisState = types.NewControllerGenesisState(activeChannels, []controllertypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate active channel - invalid channel identifier",
			func() {
				activeChannels := []controllertypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: "invalid|channel",
					},
				}

				genesisState = types.NewControllerGenesisState(activeChannels, []controllertypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate registered account - invalid port identifier",
			func() {
				activeChannels := []controllertypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []controllertypes.RegisteredInterchainAccount{
					{
						PortId:         "invalid|port",
						AccountAddress: TestOwnerAddress,
					},
				}

				genesisState = types.NewControllerGenesisState(activeChannels, registeredAccounts, []string{}, controllertypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate registered account - invalid owner address",
			func() {
				activeChannels := []controllertypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []controllertypes.RegisteredInterchainAccount{
					{
						PortId:         TestPortID,
						AccountAddress: "",
					},
				}

				genesisState = types.NewControllerGenesisState(activeChannels, registeredAccounts, []string{}, controllertypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate controller ports - invalid port identifier",
			func() {
				activeChannels := []controllertypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []controllertypes.RegisteredInterchainAccount{
					{
						PortId:         TestPortID,
						AccountAddress: TestOwnerAddress,
					},
				}

				genesisState = types.NewControllerGenesisState(activeChannels, registeredAccounts, []string{"invalid|port"}, controllertypes.DefaultParams())
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			genesisState = types.DefaultControllerGenesis()

			tc.malleate() // malleate mutates test data

			err := types.ValidateControllerGenesis(genesisState)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *TypesTestSuite) TestValidateHostGenesisState() {
	var (
		genesisState hosttypes.HostGenesisState
	)

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
				activeChannels := []hosttypes.ActiveChannel{
					{
						PortId:    "invalid|port",
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				genesisState = types.NewHostGenesisState(activeChannels, []hosttypes.RegisteredInterchainAccount{}, types.PortID, hosttypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate active channel - invalid channel identifier",
			func() {
				activeChannels := []hosttypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: "invalid|channel",
					},
				}

				genesisState = types.NewHostGenesisState(activeChannels, []hosttypes.RegisteredInterchainAccount{}, types.PortID, hosttypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate registered account - invalid port identifier",
			func() {
				activeChannels := []hosttypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []hosttypes.RegisteredInterchainAccount{
					{
						PortId:         "invalid|port",
						AccountAddress: TestOwnerAddress,
					},
				}

				genesisState = types.NewHostGenesisState(activeChannels, registeredAccounts, types.PortID, hosttypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate registered account - invalid owner address",
			func() {
				activeChannels := []hosttypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []hosttypes.RegisteredInterchainAccount{
					{
						PortId:         TestPortID,
						AccountAddress: "",
					},
				}

				genesisState = types.NewHostGenesisState(activeChannels, registeredAccounts, types.PortID, hosttypes.DefaultParams())
			},
			false,
		},
		{
			"failed to validate controller ports - invalid port identifier",
			func() {
				activeChannels := []hosttypes.ActiveChannel{
					{
						PortId:    TestPortID,
						ChannelId: ibctesting.FirstChannelID,
					},
				}

				registeredAccounts := []hosttypes.RegisteredInterchainAccount{
					{
						PortId:         TestPortID,
						AccountAddress: TestOwnerAddress,
					},
				}

				genesisState = types.NewHostGenesisState(activeChannels, registeredAccounts, "invalid|port", hosttypes.DefaultParams())
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			genesisState = types.DefaultHostGenesis()

			tc.malleate() // malleate mutates test data

			err := types.ValidateHostGenesis(genesisState)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}
