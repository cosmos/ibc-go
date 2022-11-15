package testsuite

import (
	"github.com/cosmos/cosmos-sdk/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	simappparams "github.com/cosmos/ibc-go/v6/testing/simapp/params"
)

func Codec() *codec.ProtoCodec {
	cfg := simappparams.MakeTestEncodingConfig()
	banktypes.RegisterInterfaces(cfg.InterfaceRegistry)
	govv1beta1.RegisterInterfaces(cfg.InterfaceRegistry)
	cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)
	return cdc
}
