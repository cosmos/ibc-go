//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	test "github.com/strangelove-ventures/interchaintest/v9/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "cosmossdk.io/x/gov/types"
	paramsproposaltypes "cosmossdk.io/x/params/types/proposal"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// compatibility:from_version: v7.2.0
func TestTransferTestSuiteSendEnabled(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuiteSendEnabled))
}

type TransferTestSuiteSendEnabled struct {
	transferTester
}

func (s *TransferTestSuiteSendEnabled) SetupSuite() {
	s.SetupChains(context.TODO(), nil, func(options *testsuite.ChainOptions) {
		options.RelayerCount = 1
	})
}

// TestSendEnabledParam tests changing ics20 SendEnabled parameter
func (s *TransferTestSuiteSendEnabled) TestSendEnabledParam() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	// Note: explicitly not using t.Parallel() in this test as it makes chain wide changes
	s.CreateTransferPath(testName)

	chainA, chainB := s.GetChains()

	channelA := s.GetChainAChannelForTest(testName)
	chainAVersion := chainA.Config().Images[0].Version
	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	isSelfManagingParams := testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion)

	govModuleAddress, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure transfer sending is enabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).SendEnabled
		s.Require().True(enabled)
	})

	t.Run("ensure packets can be sent", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("change send enabled parameter to disabled", func(t *testing.T) {
		if isSelfManagingParams {
			msg := transfertypes.NewMsgUpdateParams(govModuleAddress.String(), transfertypes.NewParams(false, true))
			s.ExecuteAndPassGovV1Proposal(ctx, msg, chainA, chainAWallet)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(transfertypes.StoreKey, string(transfertypes.KeySendEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainA, chainAWallet, proposal)
		}
	})

	t.Run("ensure transfer params are disabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).SendEnabled
		s.Require().False(enabled)
	})

	t.Run("ensure ics20 transfer fails", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
		s.AssertTxFailure(transferTxResp, transfertypes.ErrSendDisabled)
	})
}
