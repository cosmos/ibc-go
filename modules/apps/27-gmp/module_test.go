package gmp_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	gmp "github.com/cosmos/ibc-go/v10/modules/apps/27-gmp"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type AppModuleTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
}

func TestAppModuleTestSuite(t *testing.T) {
	testifysuite.Run(t, new(AppModuleTestSuite))
}

func (s *AppModuleTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 1)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
}

func (s *AppModuleTestSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expErr   bool
	}{
		{
			"success: default genesis",
			types.DefaultGenesisState(),
			false,
		},
		{
			"success: valid genesis with account",
			&types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: s.chainA.SenderAccount.GetAddress().String(),
						AccountId: types.AccountIdentifier{
							ClientId: ibctesting.FirstClientID,
							Sender:   s.chainA.SenderAccount.GetAddress().String(),
							Salt:     []byte("salt"),
						},
					},
				},
			},
			false,
		},
		{
			"failure: invalid account address",
			&types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: "invalid",
						AccountId: types.AccountIdentifier{
							ClientId: ibctesting.FirstClientID,
							Sender:   s.chainA.SenderAccount.GetAddress().String(),
							Salt:     []byte("salt"),
						},
					},
				},
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			module := gmp.NewAppModule(s.chainA.GetSimApp().GMPKeeper)
			cdc := s.chainA.GetSimApp().AppCodec()
			bz := cdc.MustMarshalJSON(tc.genState)

			err := module.ValidateGenesis(cdc, nil, bz)
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *AppModuleTestSuite) TestValidateGenesisInvalidJSON() {
	module := gmp.NewAppModule(s.chainA.GetSimApp().GMPKeeper)
	cdc := s.chainA.GetSimApp().AppCodec()

	err := module.ValidateGenesis(cdc, nil, json.RawMessage("invalid"))
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to unmarshal")
}

func (s *AppModuleTestSuite) TestAutoCLIOptions() {
	module := gmp.NewAppModule(s.chainA.GetSimApp().GMPKeeper)
	opts := module.AutoCLIOptions()

	s.Require().NotNil(opts)
	s.Require().NotNil(opts.Query)
	s.Require().NotNil(opts.Tx)
}

func (s *AppModuleTestSuite) TestExportGenesis() {
	module := gmp.NewAppModule(s.chainA.GetSimApp().GMPKeeper)
	cdc := s.chainA.GetSimApp().AppCodec()

	bz := module.ExportGenesis(s.chainA.GetContext(), cdc)
	s.Require().NotEmpty(bz)

	var gs types.GenesisState
	err := cdc.UnmarshalJSON(bz, &gs)
	s.Require().NoError(err)
}

func TestAppModuleName(t *testing.T) {
	module := gmp.AppModule{}
	require.Equal(t, types.ModuleName, module.Name())
}

func TestAppModuleConsensusVersion(t *testing.T) {
	module := gmp.AppModule{}
	require.Equal(t, uint64(1), module.ConsensusVersion())
}

func TestAppModuleDefaultGenesis(t *testing.T) {
	module := gmp.AppModule{}

	coordinator := ibctesting.NewCoordinator(t, 1)
	chain := coordinator.GetChain(ibctesting.GetChainID(1))
	cdc := chain.GetSimApp().AppCodec()

	bz := module.DefaultGenesis(cdc)
	require.NotEmpty(t, bz)

	var gs types.GenesisState
	err := cdc.UnmarshalJSON(bz, &gs)
	require.NoError(t, err)
	require.Empty(t, gs.Ics27Accounts)
}
