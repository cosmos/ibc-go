package types_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	controllertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/genesis/types"
	hosttypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestPortID defines a reusable port identifier for testing purposes
	TestPortID, _ = icatypes.NewControllerPortID(TestOwnerAddress)
)

type GenesisTypesTestSuite struct {
	testifysuite.Suite
}

func TestGenesisTypesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(GenesisTypesTestSuite))
}

func (suite *GenesisTypesTestSuite) TestValidateGenesisState() {
	var genesisState genesistypes.GenesisState

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failed to validate - empty value",
			func() {
				genesisState = genesistypes.GenesisState{}
			},
			host.ErrInvalidID,
		},
		{
			"failed to validate - invalid controller genesis",
			func() {
				genesisState = *genesistypes.NewGenesisState(genesistypes.ControllerGenesisState{Ports: []string{"invalid|port"}}, genesistypes.DefaultHostGenesis())
			},
			host.ErrInvalidID,
		},
		{
			"failed to validate - invalid host genesis",
			func() {
				genesisState = *genesistypes.NewGenesisState(genesistypes.DefaultControllerGenesis(), genesistypes.HostGenesisState{})
			},
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			genesisState = *genesistypes.DefaultGenesis()

			tc.malleate() // malleate mutates test data

			err := genesisState.Validate()

			if tc.expErr == nil {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *GenesisTypesTestSuite) TestValidateControllerGenesisState() {
	var genesisState genesistypes.ControllerGenesisState

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

				genesisState = genesistypes.NewControllerGenesisState(activeChannels, []genesistypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams())
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

				genesisState = genesistypes.NewControllerGenesisState(activeChannels, []genesistypes.RegisteredInterchainAccount{}, []string{}, controllertypes.DefaultParams())
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

				genesisState = genesistypes.NewControllerGenesisState(activeChannels, registeredAccounts, []string{}, controllertypes.DefaultParams())
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

				genesisState = genesistypes.NewControllerGenesisState(activeChannels, registeredAccounts, []string{}, controllertypes.DefaultParams())
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

				genesisState = genesistypes.NewControllerGenesisState(activeChannels, registeredAccounts, []string{"invalid|port"}, controllertypes.DefaultParams())
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			genesisState = genesistypes.DefaultControllerGenesis()

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

func (suite *GenesisTypesTestSuite) TestValidateHostGenesisState() {
	var genesisState genesistypes.HostGenesisState

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

				genesisState = genesistypes.NewHostGenesisState(activeChannels, []genesistypes.RegisteredInterchainAccount{}, icatypes.HostPortID, hosttypes.DefaultParams())
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

				genesisState = genesistypes.NewHostGenesisState(activeChannels, []genesistypes.RegisteredInterchainAccount{}, icatypes.HostPortID, hosttypes.DefaultParams())
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

				genesisState = genesistypes.NewHostGenesisState(activeChannels, registeredAccounts, icatypes.HostPortID, hosttypes.DefaultParams())
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

				genesisState = genesistypes.NewHostGenesisState(activeChannels, registeredAccounts, icatypes.HostPortID, hosttypes.DefaultParams())
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

				genesisState = genesistypes.NewHostGenesisState(activeChannels, registeredAccounts, "invalid|port", hosttypes.DefaultParams())
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			genesisState = genesistypes.DefaultHostGenesis()

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
