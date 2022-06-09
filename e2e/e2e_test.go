package e2e

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/ibc-go/v3/testing/simapp"
	"github.com/spf13/pflag"
	"os"
	"testing"
)

func TestSomething(t *testing.T) {
	encodingConfig := simapp.MakeTestEncodingConfig()
	initClientCtx := client.Context{}.
		//WithChainID("chain-0").
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(simapp.DefaultNodeHome).
		WithViper("") // In simapp, we don't use any prefix for env variables.

	//initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, &pflag.FlagSet{})
	//panicIfErr(err)

	initClientCtx, err := config.ReadFromClientConfig(initClientCtx)
	panicIfErr(err)

	qc := types.NewQueryClient(initClientCtx)
	resp, err := qc.Params(context.TODO(), &types.QueryParamsRequest{})
	panicIfErr(err)

	fmt.Println(resp)

	pageReq, err := client.ReadPageRequest(&pflag.FlagSet{})
	panicIfErr(err)

	accResp, err := qc.Accounts(context.TODO(), &types.QueryAccountsRequest{
		Pagination: pageReq,
	})
	panicIfErr(err)

	fmt.Println(accResp)
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
