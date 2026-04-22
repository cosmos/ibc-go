package simapp

import (
	"encoding/json"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func newStandaloneSimApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{})
	return app, app.DefaultGenesis()
}

func getStandaloneSimApp(t *testing.T, chain *ibctesting.TestChain) *SimApp {
	t.Helper()

	app, ok := chain.App.(*SimApp)
	require.True(t, ok)

	return app
}

func TestSimAppPacketForwardTransfer(t *testing.T) {
	coordinator := ibctesting.NewCustomAppCoordinator(t, 3, newStandaloneSimApp)
	chainA := coordinator.GetChain(ibctesting.GetChainID(1))
	chainB := coordinator.GetChain(ibctesting.GetChainID(2))
	chainC := coordinator.GetChain(ibctesting.GetChainID(3))

	pathAB := ibctesting.NewTransferPath(chainA, chainB)
	pathBC := ibctesting.NewTransferPath(chainB, chainC)

	pathAB.Setup()
	pathBC.Setup()

	receiverOnB := chainB.SenderAccounts[1].SenderAccount.GetAddress().String()
	receiverOnC := chainC.SenderAccounts[1].SenderAccount.GetAddress().String()
	token := sdk.NewInt64Coin(sdk.DefaultBondDenom, 100)

	memo, err := json.Marshal(packetforwardtypes.PacketMetadata{
		Forward: &packetforwardtypes.ForwardMetadata{
			Port:     transfertypes.PortID,
			Channel:  pathBC.EndpointA.ChannelID,
			Receiver: receiverOnC,
		},
	})
	require.NoError(t, err)

	msgTransfer := transfertypes.NewMsgTransfer(
		pathAB.EndpointA.ChannelConfig.PortID,
		pathAB.EndpointA.ChannelID,
		token,
		chainA.SenderAccount.GetAddress().String(),
		receiverOnB,
		chainA.GetTimeoutHeight(),
		chainA.GetTimeoutTimestamp(),
		string(memo),
	)

	res, err := chainA.SendMsgs(msgTransfer)
	require.NoError(t, err)

	packetAB, err := ibctesting.ParseV1PacketFromEvents(res.Events)
	require.NoError(t, err)

	require.NoError(t, pathAB.EndpointB.UpdateClient())

	recvABRes, err := pathAB.EndpointB.RecvPacketWithResult(packetAB)
	require.NoError(t, err)

	packetBC, err := ibctesting.ParseV1PacketFromEvents(recvABRes.Events)
	require.NoError(t, err)

	require.NoError(t, pathBC.EndpointB.UpdateClient())

	recvBCRes, err := pathBC.EndpointB.RecvPacketWithResult(packetBC)
	require.NoError(t, err)

	ackBC, err := ibctesting.ParseAckFromEvents(recvBCRes.Events)
	require.NoError(t, err)

	ackBRes, err := pathBC.EndpointA.AcknowledgePacketWithResult(packetBC, ackBC)
	require.NoError(t, err)
	require.NoError(t, pathAB.EndpointA.UpdateClient())

	ackAB, err := ibctesting.ParseAckFromEvents(ackBRes.Events)
	require.NoError(t, err)
	require.NoError(t, pathAB.EndpointA.AcknowledgePacket(packetAB, ackAB))

	chainBApp := getStandaloneSimApp(t, chainB)
	chainCApp := getStandaloneSimApp(t, chainC)

	traceAToB := transfertypes.NewHop(pathAB.EndpointB.ChannelConfig.PortID, pathAB.EndpointB.ChannelID)
	traceBToC := transfertypes.NewHop(pathBC.EndpointB.ChannelConfig.PortID, pathBC.EndpointB.ChannelID)
	chainBDenom := transfertypes.NewDenom(token.Denom, traceAToB).IBCDenom()
	chainCDenom := transfertypes.NewDenom(token.Denom, traceBToC, traceAToB).IBCDenom()

	balanceOnC := chainCApp.BankKeeper.GetBalance(chainC.GetContext(), chainC.SenderAccounts[1].SenderAccount.GetAddress(), chainCDenom)
	require.Equal(t, token.Amount, balanceOnC.Amount)

	balanceOnB := chainBApp.BankKeeper.GetBalance(chainB.GetContext(), chainB.SenderAccounts[1].SenderAccount.GetAddress(), chainBDenom)
	require.True(t, balanceOnB.IsZero())

	require.Empty(t, chainBApp.PacketForwardKeeper.ExportGenesis(chainB.GetContext()).InFlightPackets)
}
