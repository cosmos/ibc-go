package testsuite

import (
	"bytes"
	"encoding/hex"

	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/x/authz"
	banktypes "cosmossdk.io/x/bank/types"
	govv1 "cosmossdk.io/x/gov/types/v1"
	govv1beta1 "cosmossdk.io/x/gov/types/v1beta1"
	grouptypes "cosmossdk.io/x/group"
	proposaltypes "cosmossdk.io/x/params/types/proposal"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	v7migrations "github.com/cosmos/ibc-go/v9/modules/core/02-client/migrations/v7"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctmtypes "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// Codec returns the global E2E protobuf codec.
func Codec() *codec.ProtoCodec {
	cdc, _ := codecAndEncodingConfig()
	return cdc
}

// SDKEncodingConfig returns the global E2E encoding config.
func SDKEncodingConfig() *testutil.TestEncodingConfig {
	_, cfg := codecAndEncodingConfig()
	return &testutil.TestEncodingConfig{
		InterfaceRegistry: cfg.InterfaceRegistry,
		Codec:             cfg.Codec,
		TxConfig:          cfg.TxConfig,
		Amino:             cfg.Amino,
	}
}

// codecAndEncodingConfig returns the codec and encoding config used in the E2E tests.
// Note: any new types added to the codec must be added here.
func codecAndEncodingConfig() (*codec.ProtoCodec, testutil.TestEncodingConfig) {
	cfg := testutil.MakeTestEncodingConfig(codectestutil.CodecOptions{})

	// ibc types
	icacontrollertypes.RegisterInterfaces(cfg.InterfaceRegistry)
	icahosttypes.RegisterInterfaces(cfg.InterfaceRegistry)
	feetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	solomachine.RegisterInterfaces(cfg.InterfaceRegistry)
	v7migrations.RegisterInterfaces(cfg.InterfaceRegistry)
	transfertypes.RegisterInterfaces(cfg.InterfaceRegistry)
	clienttypes.RegisterInterfaces(cfg.InterfaceRegistry)
	channeltypes.RegisterInterfaces(cfg.InterfaceRegistry)
	connectiontypes.RegisterInterfaces(cfg.InterfaceRegistry)
	ibctmtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	wasmtypes.RegisterInterfaces(cfg.InterfaceRegistry)

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
	txtypes.RegisterInterfaces(cfg.InterfaceRegistry)
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

	return ibctesting.UnmarshalMsgResponses(cdc, bz, msgs...)
}

// MustProtoMarshalJSON provides an auxiliary function to return Proto3 JSON encoded
// bytes of a message. This function should be used when marshalling a proto.Message
// from the e2e tests. This function strips out unknown fields. This is useful for
// backwards compatibility tests where the types imported by the e2e package have
// new fields that older versions do not recognize.
func MustProtoMarshalJSON(msg proto.Message) []byte {
	anyResolver := codectypes.NewInterfaceRegistry()

	// EmitDefaults is set to false to prevent marshalling of unpopulated fields (memo)
	// OrigName and the anyResovler match the fields the original SDK function would expect
	// in order to minimize changes.

	// OrigName is true since there is no particular reason to use camel case
	// The any resolver is empty, but provided anyways.
	jm := &jsonpb.Marshaler{OrigName: true, EmitDefaults: false, AnyResolver: anyResolver}

	err := codectypes.UnpackInterfaces(msg, codectypes.ProtoJSONPacker{JSONPBMarshaler: jm})
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	if err := jm.Marshal(buf, msg); err != nil {
		panic(err)
	}

	return buf.Bytes()
}
