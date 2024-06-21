package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func (suite *KeeperTestSuite) TestGenesis() {
	getTrace := func(index uint) types.Trace {
		return types.NewTrace("transfer", fmt.Sprintf("channelToChain%d", index))
	}

	var (
		denoms                types.Denoms
		escrows               sdk.Coins
		traceAndEscrowAmounts = []struct {
			trace  []types.Trace
			escrow string
		}{
			{[]types.Trace{getTrace(0)}, "10"},
			{[]types.Trace{getTrace(1), getTrace(0)}, "100000"},
			{[]types.Trace{getTrace(2), getTrace(1), getTrace(0)}, "10000000000"},
			{[]types.Trace{getTrace(3), getTrace(2), getTrace(1), getTrace(0)}, "1000000000000000"},
			{[]types.Trace{getTrace(4), getTrace(3), getTrace(2), getTrace(1), getTrace(0)}, "100000000000000000000"},
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
