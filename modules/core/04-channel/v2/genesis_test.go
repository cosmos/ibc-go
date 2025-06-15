package channelv2_test

import (
	"github.com/cosmos/gogoproto/proto"

	channelv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

// TestInitExportGenesis tests the import and export flow for the channel v2 keeper.
func (s *ModuleTestSuite) TestInitExportGenesis() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupV2()

	path2 := ibctesting.NewPath(s.chainA, s.chainC)
	path2.SetupV2()

	app := s.chainA.App

	emptyGenesis := types.DefaultGenesisState()

	// create a valid genesis state that uses the client keepers existing client IDs
	clientStates := app.GetIBCKeeper().ClientKeeper.GetAllGenesisClients(s.chainA.GetContext())
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
			uint64(s.chainA.GetContext().BlockTime().Unix()),
			mockv2.NewMockPayload("src", "dst"),
		)
		bz, err := proto.Marshal(&packet)
		s.Require().NoError(err)
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
		s.Run(tt.name, func() {
			channelV2Keeper := app.GetIBCKeeper().ChannelKeeperV2

			channelv2.InitGenesis(s.chainA.GetContext(), channelV2Keeper, tt.genState)

			exported := channelv2.ExportGenesis(s.chainA.GetContext(), channelV2Keeper)
			s.Require().Equal(tt.genState, exported)
		})
	}
}
