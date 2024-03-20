//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestInterchainAccountsQueryTestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsQueryTestSuite))
}

type InterchainAccountsQueryTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *InterchainAccountsQueryTestSuite) TestInterchainAccountsQuery() {
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
	var hostAccount string

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version, channeltypes.UNORDERED)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.AssertTxSuccess(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("query via interchain account", func(t *testing.T) {
		// the host account need not be funded
		t.Run("broadcast query packet", func(t *testing.T) {
			balanceQuery := banktypes.NewQueryBalanceRequest(chainBAccount.Address(), chainB.Config().Denom)
			queryBz, err := balanceQuery.Marshal()
			s.Require().NoError(err)

			queryMsg := icahosttypes.NewMsgModuleQuerySafe(hostAccount, []*icahosttypes.QueryRequest{
				{
					Path: "/cosmos.bank.v1beta1.Query/Balance",
					Data: queryBz,
				},
			})

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{queryMsg}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			// test that it is deserializeable
			_, err = icatypes.DeserializeCosmosTx(cdc, bz, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			icaQueryMsg := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, icaQueryMsg)
			s.AssertTxSuccess(txResp)

			s.Require().NoError(testutil.WaitForBlocks(ctx, 10, chainA, chainB))
		})

		t.Run("verify query response", func(t *testing.T) {
			txSearchRes, err := s.QueryTxsByEvents(ctx, chainB, 1, 1, "message.action='/ibc.core.channel.v1.MsgRecvPacket'", "")
			s.Require().NoError(err)
			s.Require().Len(txSearchRes.Txs, 1)

			// get acknowledgement
			ackFound := false
			ack := &channeltypes.Acknowledgement{}

		search_ack:
			for _, event := range txSearchRes.Txs[0].Events {
				if event.Type != channeltypes.EventTypeWriteAck {
					continue
				}

				for _, attr := range event.Attributes {
					if attr.Key != channeltypes.AttributeKeyAckHex {
						continue
					}

					ackBz, err := hex.DecodeString(attr.Value)
					s.Require().NoError(err)

					// s.Require().Equal("", string(ackBz))

					err = json.Unmarshal(ackBz, ack)
					s.Require().NoError(err)

					ackFound = true

					break search_ack
				}
			}
			s.Require().True(ackFound)
			s.Require().NotZero(ack)
			s.Require().True(ack.Success())
			s.Require().NotZero(ack.GetResult())

			// unmarshal the ica response
			icaAck := &sdk.TxMsgData{}
			err = proto.Unmarshal(ack.GetResult(), icaAck)
			s.Require().NoError(err)
			s.Require().Equal("", icaAck.String())
			s.Require().Len(icaAck.GetMsgResponses(), 1)

			// unmarshal the tx response
			queryTxResp := &icahosttypes.MsgModuleQuerySafeResponse{}
			err = proto.Unmarshal(icaAck.MsgResponses[0].Value, queryTxResp)
			s.Require().NoError(err)
			s.Require().Len(queryTxResp.Responses, 1)

			// unmarshal the bank query response
			balanceResp := &banktypes.QueryBalanceResponse{}
			err = proto.Unmarshal(queryTxResp.Responses[0], balanceResp)
			s.Require().NoError(err)
			s.Require().Equal(balanceResp.Balance.Denom, chainB.Config().Denom)
			s.Require().Equal(balanceResp.Balance.Amount, testvalues.StartingTokenAmount)
		})
	})
}
