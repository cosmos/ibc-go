package testsuite

import (
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	simappparams "github.com/cosmos/ibc-go/v6/testing/simapp/params"
)

func Codec() *codec.ProtoCodec {
	cdc, _ := codecAndEncodingConfig()
	return cdc
}

func EncodingConfig() simappparams.EncodingConfig {
	_, cfg := codecAndEncodingConfig()
	return cfg
}

func codecAndEncodingConfig() (*codec.ProtoCodec, simappparams.EncodingConfig) {
	cfg := simappparams.MakeTestEncodingConfig()
	banktypes.RegisterInterfaces(cfg.InterfaceRegistry)
	govv1beta1.RegisterInterfaces(cfg.InterfaceRegistry)
	authtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)
	return cdc, cfg
}
