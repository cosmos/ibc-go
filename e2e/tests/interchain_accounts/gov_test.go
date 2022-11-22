package interchain_accounts

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/gogo/protobuf/proto"
	"github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func TestInterchainAccountsGovTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsGovTestSuite))
}

type InterchainAccountsGovTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *InterchainAccountsGovTestSuite) TestInterchainAccountsGovIntegration() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.Bech32Address(chainA.Config().Bech32Prefix)

	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBAccount.Bech32Address(chainB.Config().Bech32Prefix)

	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	t.Run("create msg submit proposal", func(t *testing.T) {
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, govModuleAddress.String(), version)
		msgs := []sdk.Msg{msgRegisterAccount}
		msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(sdk.NewCoin(chainA.Config().Denom, govtypesv1.DefaultMinDepositTokens)), controllerAddress, "")
		s.Require().NoError(err)

		resp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgSubmitProposal)
		s.AssertValidTxResponse(resp)
		s.Require().NoError(err)
	})

	s.Require().NoError(chainA.VoteOnProposalAllValidators(ctx, "1", cosmos.ProposalVoteYes))

	time.Sleep(testvalues.VotingPeriod)
	time.Sleep(5 * time.Second)

	proposal, err := s.QueryProposalV1(ctx, chainA, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypesv1.StatusPassed, proposal.Status)

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	var interchainAccAddr string
	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		interchainAccAddr, err = s.QueryInterchainAccount(ctx, chainA, govModuleAddress.String(), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(interchainAccAddr))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account, so it has some $$ to send
			err := chainB.SendFunds(ctx, ibctest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: interchainAccAddr,
				Amount:  testvalues.StartingTokenAmount,
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("create msg submit proposal", func(t *testing.T) {
			msgBankSend := &banktypes.MsgSend{
				FromAddress: interchainAccAddr,
				ToAddress:   chainBAddress,
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgBankSend})
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(govModuleAddress.String(), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)
			msgs := []sdk.Msg{msgSendTx}
			msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(sdk.NewCoin(chainA.Config().Denom, govtypesv1.DefaultMinDepositTokens)), controllerAddress, "")
			s.Require().NoError(err)

			resp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgSubmitProposal)
			s.AssertValidTxResponse(resp)
			s.Require().NoError(err)
		})

		s.Require().NoError(chainA.VoteOnProposalAllValidators(ctx, "2", cosmos.ProposalVoteYes))

		time.Sleep(testvalues.VotingPeriod)
		time.Sleep(5 * time.Second)

		proposal, err := s.QueryProposalV1(ctx, chainA, 2)
		s.Require().NoError(err)
		s.Require().Equal(govtypesv1.StatusPassed, proposal.Status)

		t.Run("verify tokens transferred", func(t *testing.T) {
			balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = chainB.GetBalance(ctx, interchainAccAddr, chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance)
		})
	})
}
