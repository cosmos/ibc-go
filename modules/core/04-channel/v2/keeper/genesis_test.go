package keeper_test

import (
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// TestInitExportGenesis tests the import and export flow for the channel v2 keeper.
func (suite *KeeperTestSuite) TestInitExportGenesis() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupV2()

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

		validGs.Acknowledgements = append(validGs.Acknowledgements, ack)
		validGs.Receipts = append(validGs.Receipts, receipt)
		validGs.Commitments = append(validGs.Commitments, commitment)
		validGs.SendSequences = append(validGs.SendSequences, seq)
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
			app.GetIBCKeeper().ChannelKeeperV2.InitGenesis(suite.chainA.GetContext(), tt.genState)

			exported := app.GetIBCKeeper().ChannelKeeperV2.ExportGenesis(suite.chainA.GetContext())
			suite.Require().Equal(tt.genState, exported)
		})
	}
}
