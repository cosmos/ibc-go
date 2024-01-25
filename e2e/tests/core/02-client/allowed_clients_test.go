//go:build !test_e2e

package client

import (
	"context"
	"slices"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	cmtjson "github.com/cometbft/cometbft/libs/json"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestAllowedClientsTestSuite(t *testing.T) {
	testifysuite.Run(t, new(AllowedClientsTestSuite))
}

type AllowedClientsTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *AllowedClientsTestSuite) SetupSuite() {
	chainA, chainB := s.GetChains()
	s.SetChainsIntoSuite(chainA, chainB)
}

// QueryAllowedClients queries the on-chain AllowedClients parameter for 02-client
func (s *AllowedClientsTestSuite) QueryAllowedClients(ctx context.Context, chain ibc.Chain) []string {
	queryClient := s.GetChainGRCPClients(chain).ClientQueryClient
	res, err := queryClient.ClientParams(ctx, &clienttypes.QueryClientParamsRequest{})
	s.Require().NoError(err)

	return res.Params.AllowedClients
}

// TestAllowedClientsParam tests changing the AllowedClients parameter using a governance proposal
func (s *AllowedClientsTestSuite) TestAllowedClientsParam() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	_, _ = s.SetupRelayer(ctx, s.TransferChannelOptions(), chainA, chainB)

	chainAVersion := chainA.Config().Images[0].Version

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure allowed clients are set to the default", func(t *testing.T) {
		allowedClients := s.QueryAllowedClients(ctx, chainA)

		defaultAllowedClients := clienttypes.DefaultAllowedClients
		if !testvalues.LocalhostClientFeatureReleases.IsSupported(chainAVersion) {
			defaultAllowedClients = slices.DeleteFunc(defaultAllowedClients, func(s string) bool { return s == ibcexported.Localhost })
		}
		s.Require().Equal(defaultAllowedClients, allowedClients)
	})

	allowedClient := ibcexported.Solomachine
	t.Run("change the allowed client to only allow solomachine clients", func(t *testing.T) {
		if testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion) {
			authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
			s.Require().NoError(err)
			s.Require().NotNil(authority)

			msg := clienttypes.NewMsgUpdateParams(authority.String(), clienttypes.NewParams(allowedClient))
			s.ExecuteAndPassGovV1Proposal(ctx, msg, chainA, chainAWallet)
		} else {
			value, err := cmtjson.Marshal([]string{allowedClient})
			s.Require().NoError(err)
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(ibcexported.ModuleName, string(clienttypes.KeyAllowedClients), string(value)),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainA, chainAWallet, proposal)
		}
	})

	t.Run("validate the param was successfully changed", func(t *testing.T) {
		allowedClients := s.QueryAllowedClients(ctx, chainA)
		s.Require().Equal([]string{allowedClient}, allowedClients)
	})

	t.Run("ensure querying non-allowed client's status returns Unauthorized Status", func(t *testing.T) {
		status, err := s.QueryClientStatus(ctx, chainA, ibctesting.FirstClientID)
		s.Require().NoError(err)
		s.Require().Equal(ibcexported.Unauthorized.String(), status)
	})
}
