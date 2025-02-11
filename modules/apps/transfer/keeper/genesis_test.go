package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

func (suite *KeeperTestSuite) TestGenesis() {
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
		suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), denom)

		amount, ok := sdkmath.NewIntFromString(traceAndEscrowAmount.escrow)
		suite.Require().True(ok)
		escrow := sdk.NewCoin(denom.IBCDenom(), amount)
		escrows = append(escrows, escrow)
		suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), escrow)
	}

	genesis := suite.chainA.GetSimApp().TransferKeeper.ExportGenesis(suite.chainA.GetContext())

	suite.Require().Equal(types.PortID, genesis.PortId)
	suite.Require().Equal(denoms.Sort(), genesis.Denoms)
	suite.Require().Equal(escrows.Sort(), genesis.TotalEscrowed)

	suite.Require().NotPanics(func() {
		suite.chainA.GetSimApp().TransferKeeper.InitGenesis(suite.chainA.GetContext(), *genesis)
	})

	for _, denom := range denoms {
		_, found := suite.chainA.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainA.GetContext(), denom.IBCDenom())
		suite.Require().True(found)
	}
}
