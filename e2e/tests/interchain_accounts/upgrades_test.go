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

// compatibility:from_version: v8.4.0
func TestInterchainAccountsChannelUpgradesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsChannelUpgradesTestSuite))
}

type InterchainAccountsChannelUpgradesTestSuite struct {
	testsuite.E2ETestSuite
}

// TestMsgSendTx_SuccessfulTransfer_AfterUpgradingOrdertoUnordered tests upgrading an ICA channel to
// unordered and sends a message to the host afterwards.
func (s *InterchainAccountsChannelUpgradesTestSuite) TestMsgSendTx_SuccessfulTransfer_AfterUpgradingOrdertoUnordered() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	relayer := s.CreateDefaultPaths(testName)

	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	var (
		portID      string
		hostAccount string

		initialChannelID = "channel-1"
	)

	t.Run("register interchain account", func(t *testing.T) {
		var err error
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterInterchainAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version, channeltypes.ORDERED)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterInterchainAccount)
		s.AssertTxSuccess(txResp)
		portID, err = icatypes.NewControllerPortID(controllerAddress)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = query.InterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotEmpty(hostAccount)

		_, err = query.Channel(ctx, chainA, portID, initialChannelID)
		s.Require().NoError(err)
	})

	t.Run("fund interchain account wallet", func(t *testing.T) {
		// fund the host account so it has some $$ to send
		err := chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
			Address: hostAccount,
			Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
			Denom:   chainB.Config().Denom,
		})
		s.Require().NoError(err)
	})

	t.Run("broadcast MsgSendTx", func(t *testing.T) {
		// assemble bank transfer message from host account to user account on host chain
		msgSend := &banktypes.MsgSend{
			FromAddress: hostAccount,
			ToAddress:   chainBAccount.FormattedAddress(),
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
		}

		cdc := testsuite.Codec()

		bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
		s.Require().NoError(err)

		packetData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: bz,
			Memo: "e2e",
		}

		msgSendTx := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(5*time.Minute), packetData)

		resp := s.BroadcastMessages(
			ctx,
			chainA,
			controllerAccount,
			msgSendTx,
		)

		s.AssertTxSuccess(resp)
		s.AssertPacketRelayed(ctx, chainA, portID, initialChannelID, 1)
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		balance, err := query.Balance(ctx, chainB, chainBAccount.FormattedAddress(), chainB.Config().Denom)
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance.Int64())
	})

	channel, err := query.Channel(ctx, chainA, portID, initialChannelID)
	s.Require().NoError(err)

	// upgrade the channel ordering to UNORDERED
	upgradeFields := channeltypes.NewUpgradeFields(channeltypes.UNORDERED, channel.ConnectionHops, channel.Version)

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		govModuleAddress, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
		s.Require().NoError(err)
		s.Require().NotNil(govModuleAddress)

		msg := channeltypes.NewMsgChannelUpgradeInit(portID, initialChannelID, upgradeFields, govModuleAddress.String())
		s.ExecuteAndPassGovV1Proposal(ctx, msg, chainA, controllerAccount)
	})

	t.Run("verify channel A upgraded and is now unordered", func(t *testing.T) {
		var channel channeltypes.Channel
		waitErr := test.WaitForCondition(time.Minute*2, time.Second*5, func() (bool, error) {
			channel, err = query.Channel(ctx, chainA, portID, initialChannelID)
			if err != nil {
				return false, err
			}
			return channel.Ordering == channeltypes.UNORDERED, nil
		})
		s.Require().NoErrorf(waitErr, "channel was not upgraded: expected %s got %s", channeltypes.UNORDERED, channel.Ordering)
	})

	t.Run("verify channel B upgraded and is now unordered", func(t *testing.T) {
		var channel channeltypes.Channel
		waitErr := test.WaitForCondition(time.Minute*2, time.Second*5, func() (bool, error) {
			channel, err = query.Channel(ctx, chainB, icatypes.HostPortID, initialChannelID)
			if err != nil {
				return false, err
			}
			return channel.Ordering == channeltypes.UNORDERED, nil
		})
		s.Require().NoErrorf(waitErr, "channel was not upgraded: expected %s got %s", channeltypes.UNORDERED, channel.Ordering)
	})

	t.Run("broadcast MsgSendTx", func(t *testing.T) {
		// assemble bank transfer message from host account to user account on host chain
		msgSend := &banktypes.MsgSend{
			FromAddress: hostAccount,
			ToAddress:   chainBAccount.FormattedAddress(),
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
		}

		cdc := testsuite.Codec()

		bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
		s.Require().NoError(err)

		packetData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: bz,
			Memo: "e2e",
		}

		msgSendTx := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(5*time.Minute), packetData)

		resp := s.BroadcastMessages(
			ctx,
			chainA,
			controllerAccount,
			msgSendTx,
		)

		s.AssertTxSuccess(resp)
		s.AssertPacketRelayed(ctx, chainA, portID, initialChannelID, 2)
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		balance, err := query.Balance(ctx, chainB, chainBAccount.FormattedAddress(), chainB.Config().Denom)
		s.Require().NoError(err)

		expected := 2*testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance.Int64())
	})
}
