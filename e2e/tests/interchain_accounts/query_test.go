//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// compatibility:from_version: v7.10.0
func TestInterchainAccountsQueryTestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsQueryTestSuite))
}

type InterchainAccountsQueryTestSuite struct {
	testsuite.E2ETestSuite
}

// SetupSuite sets up chains for the current test suite
func (s *InterchainAccountsQueryTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

// compatibility:TestInterchainAccountsQuery:from_versions: v7.10.0,v8.7.0,v10.0.0
func (s *InterchainAccountsQueryTestSuite) TestInterchainAccountsQuery() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)

	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version, channeltypes.UNORDERED)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.AssertTxSuccess(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = query.InterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotEmpty(hostAccount)

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Len(channels, 2)
	})

	t.Run("query via interchain account", func(t *testing.T) {
		// the host account need not be funded
		t.Run("broadcast query packet", func(t *testing.T) {
			balanceQuery := banktypes.NewQueryBalanceRequest(chainBAccount.Address(), chainB.Config().Denom)
			queryBz, err := balanceQuery.Marshal()
			s.Require().NoError(err)

			queryMsg := icahosttypes.NewMsgModuleQuerySafe(hostAccount, []icahosttypes.QueryRequest{
				{
					Path: "/cosmos.bank.v1beta1.Query/Balance",
					Data: queryBz,
				},
			})

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{queryMsg}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			icaQueryMsg := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, icaQueryMsg)
			s.AssertTxSuccess(txResp)

			s.Require().NoError(testutil.WaitForBlocks(ctx, 20, chainA, chainB))
		})

		t.Run("verify query response", func(t *testing.T) {
			var expQueryHeight uint64

			ack := &channeltypes.Acknowledgement_Result{}
			t.Run("retrieve acknowledgement", func(t *testing.T) {
				cmd := "message.action='/ibc.core.channel.v1.MsgRecvPacket'"
				txSearchRes, err := s.QueryTxsByEvents(ctx, chainB, 1, 1, cmd, "")
				s.Require().NoError(err)
				s.Require().Len(txSearchRes.TxResponses, 1)

				expQueryHeight = uint64(txSearchRes.TxResponses[0].Height)

				ackHexValue, isFound := s.ExtractValueFromEvents(
					txSearchRes.TxResponses[0].Events,
					channeltypes.EventTypeWriteAck,
					channeltypes.AttributeKeyAckHex,
				)
				s.Require().True(isFound)
				s.Require().NotEmpty(ackHexValue)

				ackBz, err := hex.DecodeString(ackHexValue)
				s.Require().NoError(err)

				err = json.Unmarshal(ackBz, ack)
				s.Require().NoError(err)
			})

			icaAck := &sdk.TxMsgData{}
			t.Run("unmarshal ica response", func(t *testing.T) {
				err := proto.Unmarshal(ack.Result, icaAck)
				s.Require().NoError(err)
				s.Require().Len(icaAck.GetMsgResponses(), 1)
			})

			queryTxResp := &icahosttypes.MsgModuleQuerySafeResponse{}
			t.Run("unmarshal MsgModuleQuerySafeResponse", func(t *testing.T) {
				err := proto.Unmarshal(icaAck.MsgResponses[0].Value, queryTxResp)
				s.Require().NoError(err)
				s.Require().Len(queryTxResp.Responses, 1)
				s.Require().Equal(expQueryHeight, queryTxResp.Height)
			})

			balanceResp := &banktypes.QueryBalanceResponse{}
			t.Run("unmarshal and verify bank query response", func(t *testing.T) {
				err := proto.Unmarshal(queryTxResp.Responses[0], balanceResp)
				s.Require().NoError(err)
				s.Require().Equal(chainB.Config().Denom, balanceResp.Balance.Denom)
				s.Require().Equal(testvalues.StartingTokenAmount, balanceResp.Balance.Amount.Int64())
			})
		})
	})
}
