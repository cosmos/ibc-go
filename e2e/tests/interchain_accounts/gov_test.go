//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// compatibility:from_version: v7.10.0
func TestInterchainAccountsGovTestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsGovTestSuite))
}

type InterchainAccountsGovTestSuite struct {
	testsuite.E2ETestSuite
}

// SetupSuite sets up chains for the current test suite
func (s *InterchainAccountsGovTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

func (s *InterchainAccountsGovTestSuite) TestInterchainAccountsGovIntegration() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)

	chainA, chainB := s.GetChains()
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBAccount.FormattedAddress()

	govModuleAddress, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	t.Run("execute proposal for MsgRegisterInterchainAccount", func(t *testing.T) {
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, govModuleAddress.String(), version, channeltypes.ORDERED)
		s.ExecuteAndPassGovV1Proposal(ctx, msgRegisterAccount, chainA, controllerAccount)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))

	var interchainAccAddr string
	t.Run("verify interchain account registration success", func(t *testing.T) {
		var err error
		interchainAccAddr, err = query.InterchainAccount(ctx, chainA, govModuleAddress.String(), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotEmpty(interchainAccAddr)

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Len(channels, 2)
	})

	t.Run("interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account, so it has some $$ to send
			err := chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: interchainAccAddr,
				Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("execute proposal for MsgSendTx", func(t *testing.T) {
			msgBankSend := &banktypes.MsgSend{
				FromAddress: interchainAccAddr,
				ToAddress:   chainBAddress,
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
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
			s.ExecuteAndPassGovV1Proposal(ctx, msgSendTx, chainA, controllerAccount)
		})

		s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB)) // wait for the ica tx to be relayed

		t.Run("verify tokens transferred", func(t *testing.T) {
			balance, err := query.Balance(ctx, chainB, chainBAccount.FormattedAddress(), chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = query.Balance(ctx, chainB, interchainAccAddr, chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance.Int64())
		})
	})
}
