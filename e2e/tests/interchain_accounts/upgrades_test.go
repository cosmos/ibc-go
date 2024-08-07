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
	controllertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

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

// TestChannelUpgrade_ICAChannelClosesAfterTimeout_Succeeds tests upgrading an ICA channel to
// wire up fee middleware and then forces it to close it by timing out a packet.
func (s *InterchainAccountsChannelUpgradesTestSuite) TestChannelUpgrade_ICAChannelClosesAfterTimeout_Succeeds() {
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

	chainBDenom := chainB.Config().Denom
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	var (
		channelID         = "channel-1"
		controllerPortID  string
		hostPortID        = icatypes.HostPortID
		interchainAccount string
		channelA          channeltypes.Channel
	)

	t.Run("register interchain account", func(t *testing.T) {
		var err error
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterInterchainAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version, channeltypes.ORDERED)
		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterInterchainAccount)
		s.AssertTxSuccess(txResp)

		controllerPortID, err = icatypes.NewControllerPortID(controllerAddress)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		interchainAccount, err = query.InterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(interchainAccount))

		channelA, err = query.Channel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)
	})

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		s.InitiateChannelUpgrade(ctx, chainA, chainAWallet, controllerPortID, channelID, s.CreateUpgradeFields(channelA))
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("verify channel A upgraded and is fee enabled", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := query.FeeEnabledChannel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

	t.Run("verify channel B upgraded and is fee enabled", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainB, hostPortID, channelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := query.FeeEnabledChannel(ctx, chainB, hostPortID, channelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

	// stop the relayer to let the submit tx message time out
	t.Run("stop relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
	})

	t.Run("submit tx message with bank transfer message that times out", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account so it has some $$ to send
			err := chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: interchainAccount,
				Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
				Denom:   chainBDenom,
			})
			s.Require().NoError(err)
		})

		t.Run("broadcast MsgSendTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: interchainAccount,
				ToAddress:   chainBAccount.FormattedAddress(),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainBDenom)),
			}

			cdc := testsuite.Codec()

			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			timeout := uint64(1)
			msgSendTx := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, timeout, packetData)

			resp := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSendTx,
			)

			s.AssertTxSuccess(resp)

			// this sleep is to allow the packet to timeout
			time.Sleep(1 * time.Second)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("verify channel A is closed due to timeout on ordered channel", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.CLOSED, channel.State, "the channel was not in an expected state")
	})

	t.Run("verify channel B is closed due to timeout on ordered channel", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainB, hostPortID, channelID)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.CLOSED, channel.State, "the channel was not in an expected state")
	})
}
