package capability_test

import (
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/ibc-go/modules/capability"
	"github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/cosmos/ibc-go/modules/capability/types"
)

func (s *CapabilityTestSuite) TestGenesis() {
	// InitGenesis must be called in order to set the intial index to 1.
	capability.InitGenesis(s.ctx, *s.keeper, *types.DefaultGenesis())

	sk1 := s.keeper.ScopeToModule(banktypes.ModuleName)
	sk2 := s.keeper.ScopeToModule(stakingtypes.ModuleName)

	cap1, err := sk1.NewCapability(s.ctx, "transfer")
	s.Require().NoError(err)
	s.Require().NotNil(cap1)

	err = sk2.ClaimCapability(s.ctx, cap1, "transfer")
	s.Require().NoError(err)

	cap2, err := sk2.NewCapability(s.ctx, "ica")
	s.Require().NoError(err)
	s.Require().NotNil(cap2)

	genState := capability.ExportGenesis(s.ctx, *s.keeper)

	newKeeper := keeper.NewKeeper(s.cdc, s.storeKey, s.memStoreKey)
	newSk1 := newKeeper.ScopeToModule(banktypes.ModuleName)
	newSk2 := newKeeper.ScopeToModule(stakingtypes.ModuleName)
	deliverCtx := s.NewTestContext()

	capability.InitGenesis(deliverCtx, *newKeeper, *genState)

	// check that all previous capabilities exist in new app after InitGenesis
	sk1Cap1, ok := newSk1.GetCapability(deliverCtx, "transfer")
	s.Require().True(ok, "could not get first capability after genesis on first ScopedKeeper")
	s.Require().Equal(*cap1, *sk1Cap1, "capability values not equal on first ScopedKeeper")

	sk2Cap1, ok := newSk2.GetCapability(deliverCtx, "transfer")
	s.Require().True(ok, "could not get first capability after genesis on first ScopedKeeper")
	s.Require().Equal(*cap1, *sk2Cap1, "capability values not equal on first ScopedKeeper")

	sk2Cap2, ok := newSk2.GetCapability(deliverCtx, "ica")
	s.Require().True(ok, "could not get second capability after genesis on second ScopedKeeper")
	s.Require().Equal(*cap2, *sk2Cap2, "capability values not equal on second ScopedKeeper")
}
