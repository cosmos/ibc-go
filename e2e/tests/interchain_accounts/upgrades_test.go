//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestInterchainAccountsChannelUpgradesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsChannelUpgradesTestSuite))
}

type InterchainAccountsChannelUpgradesTestSuite struct {
	testsuite.E2ETestSuite
}

// TestChannelUpgrade_ICAChannelClosesAfterTimeout_Succeeds tests upgrading an ICA channel to
// wire up fee middleware and then forces it to close it by timing out a packet.
func (s *InterchainAccountsChannelUpgradesTestSuite) TestChannelUpgrade_ICAChannelClosesAfterTimeout_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx, nil)
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
		msgRegisterInterchainAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)
		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterInterchainAccount)
		s.AssertTxSuccess(txResp)

		controllerPortID, err = icatypes.NewControllerPortID(controllerAddress)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		interchainAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(interchainAccount))

		channelA, err = s.QueryChannel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)
	})

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		s.initiateChannelUpgrade(ctx, chainA, chainAWallet, controllerPortID, channelID, s.createUpgradeFields(channelA))
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("verify channel A upgraded and is fee enabled", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := s.QueryFeeEnabledChannel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

	t.Run("verify channel B upgraded and is fee enabled", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainB, hostPortID, channelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := s.QueryFeeEnabledChannel(ctx, chainB, hostPortID, channelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

	// stop the relayer to let the submit tx message time out
	t.Run("stop relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
	})

	t.Run("submit tx message with bank transfer message times out", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account account so it has some $$ to send
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

			msgSendTx := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(1), packetData)

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
		s.StartRelayer(relayer)
	})

	t.Run("verify channel is closed due to timeout on ordered channel", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainA, controllerPortID, channelID)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.CLOSED, channel.State, "the channel was not in an expected state")
	})
}

// createUpgradeFields created the upgrade fields for channel
func (s *InterchainAccountsChannelUpgradesTestSuite) createUpgradeFields(channel channeltypes.Channel) channeltypes.UpgradeFields {
	versionMetadata := feetypes.Metadata{
		FeeVersion: feetypes.Version,
		AppVersion: transfertypes.Version,
	}
	versionBytes, err := feetypes.ModuleCdc.MarshalJSON(&versionMetadata)
	s.Require().NoError(err)

	return channeltypes.NewUpgradeFields(channel.Ordering, channel.ConnectionHops, string(versionBytes))
}

// initiateChannelUpgrade creates and submits a governance proposal to execute the message to initiate a channel upgrade
func (s *InterchainAccountsChannelUpgradesTestSuite) initiateChannelUpgrade(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet, portID, channelID string, upgradeFields channeltypes.UpgradeFields) {
	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chain)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	msg := channeltypes.NewMsgChannelUpgradeInit(portID, channelID, upgradeFields, govModuleAddress.String())
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, wallet)
}
