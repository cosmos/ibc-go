package testsuite

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
	proposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	v7migrations "github.com/cosmos/ibc-go/v7/modules/core/02-client/migrations/v7"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	simappparams "github.com/cosmos/ibc-go/v7/testing/simapp/params"
)

// Codec returns the global E2E protobuf codec.
func Codec() *codec.ProtoCodec {
	cdc, _ := codecAndEncodingConfig()
	return cdc
}

// EncodingConfig returns the global E2E encoding config.
func EncodingConfig() simappparams.EncodingConfig {
	_, cfg := codecAndEncodingConfig()
	return cfg
}

func codecAndEncodingConfig() (*codec.ProtoCodec, simappparams.EncodingConfig) {
	cfg := simappparams.MakeTestEncodingConfig()

	// ibc types
	icacontrollertypes.RegisterInterfaces(cfg.InterfaceRegistry)
	feetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	intertxtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	solomachine.RegisterInterfaces(cfg.InterfaceRegistry)
	v7migrations.RegisterInterfaces(cfg.InterfaceRegistry)
	transfertypes.RegisterInterfaces(cfg.InterfaceRegistry)
	clienttypes.RegisterInterfaces(cfg.InterfaceRegistry)
	channeltypes.RegisterInterfaces(cfg.InterfaceRegistry)
	connectiontypes.RegisterInterfaces(cfg.InterfaceRegistry)
	ibctmtypes.RegisterInterfaces(cfg.InterfaceRegistry)

	// all other types
	upgradetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	banktypes.RegisterInterfaces(cfg.InterfaceRegistry)
	govv1beta1.RegisterInterfaces(cfg.InterfaceRegistry)
	govv1.RegisterInterfaces(cfg.InterfaceRegistry)
	authtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	cryptocodec.RegisterInterfaces(cfg.InterfaceRegistry)
	grouptypes.RegisterInterfaces(cfg.InterfaceRegistry)
	proposaltypes.RegisterInterfaces(cfg.InterfaceRegistry)
	authz.RegisterInterfaces(cfg.InterfaceRegistry)

	cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)
	return cdc, cfg
}

// UnmarshalMsgResponses attempts to unmarshal the tx msg responses into the provided message types.
func UnmarshalMsgResponses(txResp sdk.TxResponse, msgs ...codec.ProtoMarshaler) error {
	cdc := Codec()
	bz, err := hex.DecodeString(txResp.Data)
	if err != nil {
		return err
	}

	var txMsgData sdk.TxMsgData
	if err := cdc.Unmarshal(bz, &txMsgData); err != nil {
		return err
	}

	if len(msgs) != len(txMsgData.MsgResponses) {
		return fmt.Errorf("expected %d message responses but got %d", len(msgs), len(txMsgData.MsgResponses))
	}

	for i, msg := range msgs {
		if err := cdc.Unmarshal(txMsgData.MsgResponses[i].Value, msg); err != nil {
			return err
		}
	}

	return nil
}
