package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func (suite *KeeperTestSuite) TestGenesis() {
	const prefix = "transfer/channelToChain"
	var (
		traces                types.Traces
		escrows               sdk.Coins
		pathsAndEscrowAmounts = []struct {
			path   string
			escrow string
		}{
			{fmt.Sprintf("%s%d", prefix, 0), "10"},
			{fmt.Sprintf("%s%d/%s%d", prefix, 1, prefix, 0), "100000"},
			{fmt.Sprintf("%s%d/%s%d/%s%d", prefix, 1, prefix, 1, prefix, 0), "10000000000"},
			{fmt.Sprintf("%s%d/%s%d/%s%d/%s%d", prefix, 3, prefix, 2, prefix, 1, prefix, 0), "1000000000000000"},
			{fmt.Sprintf("%s%d/%s%d/%s%d/%s%d/%s%d", prefix, 4, prefix, 3, prefix, 2, prefix, 1, prefix, 0), "100000000000000000000"},
		}
	)

	for _, pathAndEscrowMount := range pathsAndEscrowAmounts {
		denomTrace := types.DenomTrace{
			BaseDenom: "uatom",
			Path:      pathAndEscrowMount.path,
		}
		traces = append(types.Traces{denomTrace}, traces...)
		suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(suite.chainA.GetContext(), denomTrace)

		denom := denomTrace.IBCDenom()
		amount, ok := math.NewIntFromString(pathAndEscrowMount.escrow)
		suite.Require().True(ok)
		escrows = append(sdk.NewCoins(sdk.NewCoin(denom, amount)), escrows...)
		suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), denom, amount)
	}

	genesis := suite.chainA.GetSimApp().TransferKeeper.ExportGenesis(suite.chainA.GetContext())

	suite.Require().Equal(types.PortID, genesis.PortId)
	suite.Require().Equal(traces.Sort(), genesis.DenomTraces)
	suite.Require().Equal(escrows.Sort(), genesis.DenomEscrows)

	suite.Require().NotPanics(func() {
		suite.chainA.GetSimApp().TransferKeeper.InitGenesis(suite.chainA.GetContext(), *genesis)
	})
}
