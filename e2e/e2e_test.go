package e2e

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/ibc-go/v3/testing/simapp"
	dockerClient "github.com/docker/docker/client"
	"github.com/spf13/pflag"
	"os"
	"strings"
	"testing"
)

const (
	TestChain1ID      = "test-1"
	TestChain2ID      = "test-2"
	TestContainerName = "simd-docker-chain-1-1"
)

func TestTokenTransfer(t *testing.T) {
	// This test assumes a testing environment is available already.
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	if err != nil {
		panic(err)
	}
	cmdString := "simd q ibc channel channels --home ./data/test-1 --node tcp://localhost:16657"
	//cmdString := "simd tx ibc-transfer transfer transfer channel-0 $WALLET_3 1000stake --from $WALLET_1 --home ./data/test-1 --keyring-backend test --chain-id test-1 --node tcp://localhost:16657"

	command := strings.Split(cmdString, " ")

	res, err := Exec(context.TODO(), cli, TestContainerName, command)
	panicIfErr(err)

	out := res.Combined()
	fmt.Println(out)
}

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
		fmt.Println(err)
		panic(err)
	}
}
