package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func (suite *KeeperTestSuite) TestGenesis() {
	var (
		path    string
		traces  types.Traces
		escrows types.Escrows
	)

	for i := 0; i < 5; i++ {
		prefix := fmt.Sprintf("transfer/channelToChain%d", i)
		if i == 0 {
			path = prefix
		} else {
			path = prefix + "/" + path
		}

		denomTrace := types.DenomTrace{
			BaseDenom: "uatom",
			Path:      path,
		}
		traces = append(types.Traces{denomTrace}, traces...)
		suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(suite.chainA.GetContext(), denomTrace)

		denom := denomTrace.IBCDenom()
		totalEscrow := sdkmath.NewInt(100)
		escrows = append(types.Escrows{types.DenomEscrow{Denom: denom, TotalEscrow: totalEscrow.Int64()}}, escrows...)
		suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), denom, totalEscrow)
	}

	genesis := suite.chainA.GetSimApp().TransferKeeper.ExportGenesis(suite.chainA.GetContext())

	suite.Require().Equal(types.PortID, genesis.PortId)
	suite.Require().Equal(traces.Sort(), genesis.DenomTraces)
	suite.Require().Equal(escrows.Sort(), genesis.DenomEscrows)

	suite.Require().NotPanics(func() {
		suite.chainA.GetSimApp().TransferKeeper.InitGenesis(suite.chainA.GetContext(), *genesis)
	})
}
