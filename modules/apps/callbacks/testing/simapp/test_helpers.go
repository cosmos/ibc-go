package simapp

import (
	"math/rand"
	"time"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SignAndDeliver signs and delivers a transaction. No simulation occurs as the
// ibc testing package causes checkState and deliverState to diverge in block time.
//
// CONTRACT: BeginBlock must be called before this function.
func SignAndDeliver(
	txCfg client.TxConfig, app *bam.BaseApp, msgs []sdk.Msg,
	chainID string, accNums, accSeqs []uint64, priv ...cryptotypes.PrivKey,
) (sdk.GasInfo, *sdk.Result, error) {
	tx, err := simtestutil.GenSignedMockTx(
		rand.New(rand.NewSource(time.Now().UnixNano())),
		txCfg,
		msgs,
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)},
		simtestutil.DefaultGenTxGas,
		chainID,
		accNums,
		accSeqs,
		priv...,
	)
	if err != nil {
		return sdk.GasInfo{}, nil, err
	}

	return app.SimDeliver(txCfg.TxEncoder(), tx)
}
