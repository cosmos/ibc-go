//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (s *InterchainAccountsTestSuite) TestInterchainAccountsGovIntegration() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	_, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
	s.Require().NoError(err)

	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)

	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, s.chainB)
	chainBAddress := chainBAccount.FormattedAddress()

	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, s.chainA)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	t.Run("execute proposal for MsgRegisterInterchainAccount", func(t *testing.T) {
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, govModuleAddress.String(), version)
		s.ExecuteAndPassGovV1Proposal(ctx, msgRegisterAccount, s.chainA, controllerAccount)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(s.rly)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, s.chainA, s.chainB))

	var interchainAccAddr string
	t.Run("verify interchain account registration success", func(t *testing.T) {
		var err error
		interchainAccAddr, err = s.QueryInterchainAccount(ctx, s.chainA, govModuleAddress.String(), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(interchainAccAddr))

		channels, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
		chanNumber++
		s.Require().NoError(err)
		s.Require().Equal(len(channels), chanNumber)
	})

	t.Run("interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account, so it has some $$ to send
			err := s.chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: interchainAccAddr,
				Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
				Denom:   s.chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("execute proposal for MsgSendTx", func(t *testing.T) {
			msgBankSend := &banktypes.MsgSend{
				FromAddress: interchainAccAddr,
				ToAddress:   chainBAddress,
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(s.chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgBankSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(govModuleAddress.String(), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)
			s.ExecuteAndPassGovV1Proposal(ctx, msgSendTx, s.chainA, controllerAccount)
			s.Require().NoError(test.WaitForBlocks(ctx, 5, s.chainA, s.chainB))
		})

		t.Run("verify tokens transferred", func(t *testing.T) {
			balance, err := s.QueryBalance(ctx, s.chainB, chainBAccount.FormattedAddress(), s.chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = s.QueryBalance(ctx, s.chainB, interchainAccAddr, s.chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance.Int64())
		})
	})
}
