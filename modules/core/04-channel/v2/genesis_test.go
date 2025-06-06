package channelv2_test

import (
	"github.com/cosmos/gogoproto/proto"

	channelv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

// TestInitExportGenesis tests the import and export flow for the channel v2 keeper.
func (suite *ModuleTestSuite) TestInitExportGenesis() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupV2()

	path2 := ibctesting.NewPath(suite.chainA, suite.chainC)
	path2.SetupV2()

	app := suite.chainA.App

	emptyGenesis := types.DefaultGenesisState()

	// create a valid genesis state that uses the client keepers existing client IDs
	clientStates := app.GetIBCKeeper().ClientKeeper.GetAllGenesisClients(suite.chainA.GetContext())
	validGs := types.DefaultGenesisState()
	for i, clientState := range clientStates {
		ack := types.NewPacketState(clientState.ClientId, uint64(i+1), []byte("ack"))
		receipt := types.NewPacketState(clientState.ClientId, uint64(i+1), []byte{byte(0x2)})
		commitment := types.NewPacketState(clientState.ClientId, uint64(i+1), []byte("commit_hash"))
		seq := types.NewPacketSequence(clientState.ClientId, uint64(i+1))

		packet := types.NewPacket(
			uint64(i+1),
			clientState.ClientId,
			clientState.ClientId,
			uint64(suite.chainA.GetContext().BlockTime().Unix()),
			mockv2.NewMockPayload("src", "dst"),
		)
		bz, err := proto.Marshal(&packet)
		suite.Require().NoError(err)
		asyncPacket := types.NewPacketState(clientState.ClientId, uint64(i+1), bz)

		validGs.Acknowledgements = append(validGs.Acknowledgements, ack)
		validGs.Receipts = append(validGs.Receipts, receipt)
		validGs.Commitments = append(validGs.Commitments, commitment)
		validGs.SendSequences = append(validGs.SendSequences, seq)
		validGs.AsyncPackets = append(validGs.AsyncPackets, asyncPacket)
		emptyGenesis.SendSequences = append(emptyGenesis.SendSequences, seq)
	}

	tests := []struct {
		name     string
		genState types.GenesisState
	}{
		{
			name:     "no modifications genesis",
			genState: emptyGenesis,
		},
		{
			name:     "valid",
			genState: validGs,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			channelV2Keeper := app.GetIBCKeeper().ChannelKeeperV2

			channelv2.InitGenesis(suite.chainA.GetContext(), channelV2Keeper, tt.genState)

			exported := channelv2.ExportGenesis(suite.chainA.GetContext(), channelV2Keeper)
			suite.Require().Equal(tt.genState, exported)
		})
	}
}
