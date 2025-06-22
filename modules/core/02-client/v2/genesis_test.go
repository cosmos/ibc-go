package clientv2_test

import (
	clientv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// TestInitExportGenesis tests the import and export flow for the channel v2 keeper.
func (s *ModuleTestSuite) TestInitExportGenesis() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupV2()

	path2 := ibctesting.NewPath(s.chainA, s.chainC)
	path2.SetupV2()

	path3 := ibctesting.NewPath(s.chainB, s.chainC)
	path3.SetupV2()

	app := s.chainA.App

	emptyGenesis := types.DefaultGenesisState()

	// create a valid genesis state that uses the counterparty info set during setup
	existingGS := clientv2.ExportGenesis(s.chainA.GetContext(), app.GetIBCKeeper().ClientV2Keeper)

	tests := []struct {
		name          string
		genState      types.GenesisState
		expectedState types.GenesisState
	}{
		{
			name:          "no modifications genesis",
			genState:      emptyGenesis,
			expectedState: existingGS,
		},
		{
			name:          "valid - default genesis",
			genState:      types.DefaultGenesisState(),
			expectedState: existingGS,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			clientV2Keeper := app.GetIBCKeeper().ClientV2Keeper

			clientv2.InitGenesis(s.chainA.GetContext(), clientV2Keeper, tt.genState)

			exported := clientv2.ExportGenesis(s.chainA.GetContext(), clientV2Keeper)
			s.Require().Equal(tt.expectedState, exported)
		})
	}
}
