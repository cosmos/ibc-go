package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/icq/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	var (
		path       *ibctesting.Path
		packetData []byte
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"icq successfully queries banktypes.AllBalances",
			func() {
				q := banktypes.QueryAllBalancesRequest{
					Address: suite.chainA.SenderAccount.GetAddress().String(),
					Pagination: &query.PageRequest{
						Offset: 0,
						Limit:  10,
					},
				}
				reqs := []abcitypes.RequestQuery{
					{
						Path: "/cosmos.bank.v1beta1.Query/AllBalances",
						Data: suite.chainA.GetSimApp().AppCodec().MustMarshal(&q),
					},
				}
				data, err := types.SerializeCosmosQuery(reqs)
				suite.Require().NoError(err)

				icqPacketData := types.InterchainQueryPacketData{
					Data: data,
				}
				packetData = icqPacketData.GetBytes()

				params := types.NewParams(true, []string{"/cosmos.bank.v1beta1.Query/AllBalances"})
				suite.chainB.GetSimApp().ICQKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"cannot unmarshal interchain query packet data",
			func() {
				packetData = []byte{}
			},
			false,
		},
		{
			"cannot deserialize interchain query packet data messages",
			func() {
				data := []byte("invalid packet data")

				icaPacketData := types.InterchainQueryPacketData{
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: message type not allowed", // NOTE: do not update params to explicitly force the error
			func() {
				q := banktypes.QueryAllBalancesRequest{}
				reqs := []abcitypes.RequestQuery{
					{
						Path: "/cosmos.bank.v1beta1.Query/AllBalances",
						Data: suite.chainA.GetSimApp().AppCodec().MustMarshal(&q),
					},
				}
				data, err := types.SerializeCosmosQuery(reqs)
				suite.Require().NoError(err)

				icaPacketData := types.InterchainQueryPacketData{
					Data: data,
				}
				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: can not perform historical query (i.e. height != 0)",
			func() {
				q := banktypes.QueryAllBalancesRequest{}
				reqs := []abcitypes.RequestQuery{
					{
						Path:   "/cosmos.bank.v1beta1.Query/AllBalances",
						Data:   suite.chainA.GetSimApp().AppCodec().MustMarshal(&q),
						Height: 1,
					},
				}
				data, err := types.SerializeCosmosQuery(reqs)
				suite.Require().NoError(err)

				icaPacketData := types.InterchainQueryPacketData{
					Data: data,
				}
				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"/cosmos.bank.v1beta1.Query/AllBalances"})
				suite.chainB.GetSimApp().ICQKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			false,
		},
		{
			"unauthorised: can not fetch query proof (i.e. prove == true)",
			func() {
				q := banktypes.QueryAllBalancesRequest{}
				reqs := []abcitypes.RequestQuery{
					{
						Path:  "/cosmos.bank.v1beta1.Query/AllBalances",
						Data:  suite.chainA.GetSimApp().AppCodec().MustMarshal(&q),
						Prove: true,
					},
				}
				data, err := types.SerializeCosmosQuery(reqs)
				suite.Require().NoError(err)

				icaPacketData := types.InterchainQueryPacketData{
					Data: data,
				}
				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"/cosmos.bank.v1beta1.Query/AllBalances"})
				suite.chainB.GetSimApp().ICQKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			path = NewICQPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICQPath(path)
			suite.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			packet := channeltypes.NewPacket(
				packetData,
				suite.chainA.SenderAccount.GetSequence(),
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(1, 100),
				0,
			)

			txResponse, err := suite.chainB.GetSimApp().ICQKeeper.OnRecvPacket(suite.chainB.GetContext(), packet)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(txResponse)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(txResponse)
			}
		})
	}
}
