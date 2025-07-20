package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func (s *KeeperTestSuite) TestGenesis() {
	getHop := func(index uint) types.Hop {
		return types.NewHop("transfer", fmt.Sprintf("channelToChain%d", index))
	}

	var (
		denoms                types.Denoms
		escrows               sdk.Coins
		traceAndEscrowAmounts = []struct {
			trace  []types.Hop
			escrow string
		}{
			{[]types.Hop{getHop(0)}, "10"},
			{[]types.Hop{getHop(1), getHop(0)}, "100000"},
			{[]types.Hop{getHop(2), getHop(1), getHop(0)}, "10000000000"},
			{[]types.Hop{getHop(3), getHop(2), getHop(1), getHop(0)}, "1000000000000000"},
			{[]types.Hop{getHop(4), getHop(3), getHop(2), getHop(1), getHop(0)}, "100000000000000000000"},
		}
	)

	for _, traceAndEscrowAmount := range traceAndEscrowAmounts {
		denom := types.NewDenom("uatom", traceAndEscrowAmount.trace...)
		denoms = append(denoms, denom)
		s.chainA.GetSimApp().TransferKeeper.SetDenom(s.chainA.GetContext(), denom)

		amount, ok := sdkmath.NewIntFromString(traceAndEscrowAmount.escrow)
		s.Require().True(ok)
		escrow := sdk.NewCoin(denom.IBCDenom(), amount)
		escrows = append(escrows, escrow)
		s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), escrow)
	}

	genesis := s.chainA.GetSimApp().TransferKeeper.ExportGenesis(s.chainA.GetContext())

	s.Require().Equal(types.PortID, genesis.PortId)
	s.Require().Equal(denoms.Sort(), genesis.Denoms)
	s.Require().Equal(escrows.Sort(), genesis.TotalEscrowed)

	s.Require().NotPanics(func() {
		s.chainA.GetSimApp().TransferKeeper.InitGenesis(s.chainA.GetContext(), *genesis)
	})

	for _, denom := range denoms {
		_, found := s.chainA.GetSimApp().BankKeeper.GetDenomMetaData(s.chainA.GetContext(), denom.IBCDenom())
		s.Require().True(found)
	}
}
