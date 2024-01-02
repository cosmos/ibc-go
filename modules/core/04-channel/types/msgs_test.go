package types_test

import (
	"fmt"
	"testing"
	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	log "cosmossdk.io/log"
	"cosmossdk.io/store/iavl"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/cosmos/ibc-go/v8/testing/mock"
	"github.com/cosmos/ibc-go/v8/testing/simapp"
)

const (
	// valid constants used for testing
	portid                      = "testportid"
	chanid                      = "channel-0"
	cpportid                    = "testcpport"
	cpchanid                    = "testcpchannel"
	counterpartyUpgradeSequence = 0

	version = "1.0"

	// invalid constants used for testing
	invalidPort      = "(invalidport1)"
	invalidShortPort = "p"
	// 195 characters
	invalidLongPort = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis eros neque, ultricies vel ligula ac, convallis porttitor elit. Maecenas tincidunt turpis elit, vel faucibus nisl pellentesque sodales"

	invalidChannel      = "(invalidchannel1)"
	invalidShortChannel = "invalid"
	invalidLongChannel  = "invalidlongchannelinvalidlongchannelinvalidlongchannelinvalidlongchannel"

	invalidConnection      = "(invalidconnection1)"
	invalidShortConnection = "invalidcn"
	invalidLongConnection  = "invalidlongconnectioninvalidlongconnectioninvalidlongconnectioninvalid"
)

// define variables used for testing
var (
	height            = clienttypes.NewHeight(0, 1)
	timeoutHeight     = clienttypes.NewHeight(0, 100)
	timeoutTimestamp  = uint64(100)
	disabledTimeout   = clienttypes.ZeroHeight()
	validPacketData   = []byte("testdata")
	unknownPacketData = []byte("unknown")

	packet        = types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp)
	invalidPacket = types.NewPacket(unknownPacketData, 0, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp)

	emptyProof = []byte{}

	addr      = sdk.AccAddress("testaddr111111111111").String()
	emptyAddr string

	connHops             = []string{"testconnection"}
	invalidConnHops      = []string{"testconnection", "testconnection"}
	invalidShortConnHops = []string{invalidShortConnection}
	invalidLongConnHops  = []string{invalidLongConnection}
)

type TypesTestSuite struct {
	testifysuite.Suite

	proof []byte
}

func (suite *TypesTestSuite) SetupTest() {
	app := simapp.Setup(suite.T(), false)
	db := dbm.NewMemDB()
	store := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	storeKey := storetypes.NewKVStoreKey("iavlStoreKey")

	store.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
	err := store.LoadVersion(0)
	suite.Require().NoError(err)
	iavlStore := store.GetCommitStore(storeKey).(*iavl.Store)

	iavlStore.Set([]byte("KEY"), []byte("VALUE"))
	_ = store.Commit()

	res, err := store.Query(&storetypes.RequestQuery{
		Data:  []byte("KEY"),
		Path:  fmt.Sprintf("/%s/key", storeKey.Name()), // required path to get key/value+proof
		Prove: true,
	})
	suite.Require().NoError(err)

	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	suite.Require().NoError(err)
	proof, err := app.AppCodec().Marshal(&merkleProof)
	suite.Require().NoError(err)

	suite.proof = proof
}

func TestTypesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestMsgChannelOpenInitValidateBasic() {
	counterparty := types.NewCounterparty(cpportid, cpchanid)
	tryOpenChannel := types.NewChannel(types.TRYOPEN, types.ORDERED, counterparty, connHops, version)

	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenInit
		expPass bool
	}{
		{"", types.NewMsgChannelOpenInit(portid, version, types.ORDERED, connHops, cpportid, addr), true},
		{"too short port id", types.NewMsgChannelOpenInit(invalidShortPort, version, types.ORDERED, connHops, cpportid, addr), false},
		{"too long port id", types.NewMsgChannelOpenInit(invalidLongPort, version, types.ORDERED, connHops, cpportid, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenInit(invalidPort, version, types.ORDERED, connHops, cpportid, addr), false},
		{"invalid channel order", types.NewMsgChannelOpenInit(portid, version, types.Order(3), connHops, cpportid, addr), false},
		{"connection hops more than 1 ", types.NewMsgChannelOpenInit(portid, version, types.ORDERED, invalidConnHops, cpportid, addr), false},
		{"too short connection id", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidShortConnHops, cpportid, addr), false},
		{"too long connection id", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidLongConnHops, cpportid, addr), false},
		{"connection id contains non-alpha", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, []string{invalidConnection}, cpportid, addr), false},
		{"", types.NewMsgChannelOpenInit(portid, "", types.UNORDERED, connHops, cpportid, addr), true},
		{"invalid counterparty port id", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, connHops, invalidPort, addr), false},
		{"channel not in INIT state", &types.MsgChannelOpenInit{portid, tryOpenChannel, addr}, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelOpenInitGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelOpenInit(portid, version, types.ORDERED, connHops, cpportid, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelOpenTryValidateBasic() {
	counterparty := types.NewCounterparty(cpportid, cpchanid)
	initChannel := types.NewChannel(types.INIT, types.ORDERED, counterparty, connHops, version)

	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenTry
		expPass bool
	}{
		{"", types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), true},
		{"too short port id", types.NewMsgChannelOpenTry(invalidShortPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"too long port id", types.NewMsgChannelOpenTry(invalidLongPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenTry(invalidPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"", types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, "", suite.proof, height, addr), true},
		{"invalid channel order", types.NewMsgChannelOpenTry(portid, version, types.Order(4), connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"connection hops more than 1 ", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidConnHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"too short connection id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidShortConnHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"too long connection id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidLongConnHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"connection id contains non-alpha", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, []string{invalidConnection}, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"", types.NewMsgChannelOpenTry(portid, "", types.UNORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), true},
		{"invalid counterparty port id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, invalidPort, cpchanid, version, suite.proof, height, addr), false},
		{"invalid counterparty channel id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, cpportid, invalidChannel, version, suite.proof, height, addr), false},
		{"empty proof", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, cpportid, cpchanid, version, emptyProof, height, addr), false},
		{"channel not in TRYOPEN state", &types.MsgChannelOpenTry{portid, "", initChannel, version, suite.proof, height, addr}, false},
		{"previous channel id is not empty", &types.MsgChannelOpenTry{portid, chanid, initChannel, version, suite.proof, height, addr}, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelOpenTryGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelOpenAckValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenAck
		expPass bool
	}{
		{"", types.NewMsgChannelOpenAck(portid, chanid, chanid, version, suite.proof, height, addr), true},
		{"too short port id", types.NewMsgChannelOpenAck(invalidShortPort, chanid, chanid, version, suite.proof, height, addr), false},
		{"too long port id", types.NewMsgChannelOpenAck(invalidLongPort, chanid, chanid, version, suite.proof, height, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenAck(invalidPort, chanid, chanid, version, suite.proof, height, addr), false},
		{"too short channel id", types.NewMsgChannelOpenAck(portid, invalidShortChannel, chanid, version, suite.proof, height, addr), false},
		{"too long channel id", types.NewMsgChannelOpenAck(portid, invalidLongChannel, chanid, version, suite.proof, height, addr), false},
		{"channel id contains non-alpha", types.NewMsgChannelOpenAck(portid, invalidChannel, chanid, version, suite.proof, height, addr), false},
		{"", types.NewMsgChannelOpenAck(portid, chanid, chanid, "", suite.proof, height, addr), true},
		{"empty proof", types.NewMsgChannelOpenAck(portid, chanid, chanid, version, emptyProof, height, addr), false},
		{"invalid counterparty channel id", types.NewMsgChannelOpenAck(portid, chanid, invalidShortChannel, version, suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelOpenAckGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelOpenAck(portid, chanid, chanid, version, suite.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelOpenConfirmValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenConfirm
		expPass bool
	}{
		{"", types.NewMsgChannelOpenConfirm(portid, chanid, suite.proof, height, addr), true},
		{"too short port id", types.NewMsgChannelOpenConfirm(invalidShortPort, chanid, suite.proof, height, addr), false},
		{"too long port id", types.NewMsgChannelOpenConfirm(invalidLongPort, chanid, suite.proof, height, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenConfirm(invalidPort, chanid, suite.proof, height, addr), false},
		{"too short channel id", types.NewMsgChannelOpenConfirm(portid, invalidShortChannel, suite.proof, height, addr), false},
		{"too long channel id", types.NewMsgChannelOpenConfirm(portid, invalidLongChannel, suite.proof, height, addr), false},
		{"channel id contains non-alpha", types.NewMsgChannelOpenConfirm(portid, invalidChannel, suite.proof, height, addr), false},
		{"empty proof", types.NewMsgChannelOpenConfirm(portid, chanid, emptyProof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelOpenConfirmGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelOpenConfirm(portid, chanid, suite.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelCloseInitValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelCloseInit
		expPass bool
	}{
		{"", types.NewMsgChannelCloseInit(portid, chanid, addr), true},
		{"too short port id", types.NewMsgChannelCloseInit(invalidShortPort, chanid, addr), false},
		{"too long port id", types.NewMsgChannelCloseInit(invalidLongPort, chanid, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelCloseInit(invalidPort, chanid, addr), false},
		{"too short channel id", types.NewMsgChannelCloseInit(portid, invalidShortChannel, addr), false},
		{"too long channel id", types.NewMsgChannelCloseInit(portid, invalidLongChannel, addr), false},
		{"channel id contains non-alpha", types.NewMsgChannelCloseInit(portid, invalidChannel, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelCloseInitGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelCloseInit(portid, chanid, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelCloseConfirmValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelCloseConfirm
		expPass bool
	}{
		{"success", types.NewMsgChannelCloseConfirm(portid, chanid, suite.proof, height, addr, 0), true},
		{"success, positive counterparty upgrade sequence", types.NewMsgChannelCloseConfirm(portid, chanid, suite.proof, height, addr, 1), true},
		{"too short port id", types.NewMsgChannelCloseConfirm(invalidShortPort, chanid, suite.proof, height, addr, 0), false},
		{"too long port id", types.NewMsgChannelCloseConfirm(invalidLongPort, chanid, suite.proof, height, addr, 0), false},
		{"port id contains non-alpha", types.NewMsgChannelCloseConfirm(invalidPort, chanid, suite.proof, height, addr, 0), false},
		{"too short channel id", types.NewMsgChannelCloseConfirm(portid, invalidShortChannel, suite.proof, height, addr, 0), false},
		{"too long channel id", types.NewMsgChannelCloseConfirm(portid, invalidLongChannel, suite.proof, height, addr, 0), false},
		{"channel id contains non-alpha", types.NewMsgChannelCloseConfirm(portid, invalidChannel, suite.proof, height, addr, 0), false},
		{"empty proof", types.NewMsgChannelCloseConfirm(portid, chanid, emptyProof, height, addr, 0), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelCloseConfirmGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelCloseConfirm(portid, chanid, suite.proof, height, addr, counterpartyUpgradeSequence)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgRecvPacketValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgRecvPacket
		expPass bool
	}{
		{"success", types.NewMsgRecvPacket(packet, suite.proof, height, addr), true},
		{"missing signer address", types.NewMsgRecvPacket(packet, suite.proof, height, emptyAddr), false},
		{"proof contain empty proof", types.NewMsgRecvPacket(packet, emptyProof, height, addr), false},
		{"invalid packet", types.NewMsgRecvPacket(invalidPacket, suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.NoError(err)
			} else {
				suite.Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgRecvPacketGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgRecvPacket(packet, suite.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgTimeoutValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgTimeout
		expPass bool
	}{
		{"success", types.NewMsgTimeout(packet, 1, suite.proof, height, addr), true},
		{"seq 0", types.NewMsgTimeout(packet, 0, suite.proof, height, addr), false},
		{"missing signer address", types.NewMsgTimeout(packet, 1, suite.proof, height, emptyAddr), false},
		{"cannot submit an empty proof", types.NewMsgTimeout(packet, 1, emptyProof, height, addr), false},
		{"invalid packet", types.NewMsgTimeout(invalidPacket, 1, suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgTimeoutGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgTimeout(packet, 1, suite.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgTimeoutOnCloseValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgTimeoutOnClose
		expPass bool
	}{
		{"success", types.NewMsgTimeoutOnClose(packet, 1, suite.proof, suite.proof, height, addr, 0), true},
		{"success, positive counterparty upgrade sequence", types.NewMsgTimeoutOnClose(packet, 1, suite.proof, suite.proof, height, addr, 1), true},
		{"seq 0", types.NewMsgTimeoutOnClose(packet, 0, suite.proof, suite.proof, height, addr, 0), false},
		{"signer address is empty", types.NewMsgTimeoutOnClose(packet, 1, suite.proof, suite.proof, height, emptyAddr, 0), false},
		{"empty proof", types.NewMsgTimeoutOnClose(packet, 1, emptyProof, suite.proof, height, addr, 0), false},
		{"empty proof close", types.NewMsgTimeoutOnClose(packet, 1, suite.proof, emptyProof, height, addr, 0), false},
		{"invalid packet", types.NewMsgTimeoutOnClose(invalidPacket, 1, suite.proof, suite.proof, height, addr, 0), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgTimeoutOnCloseGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgTimeoutOnClose(packet, 1, suite.proof, suite.proof, height, addr, counterpartyUpgradeSequence)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgAcknowledgementValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgAcknowledgement
		expPass bool
	}{
		{"success", types.NewMsgAcknowledgement(packet, packet.GetData(), suite.proof, height, addr), true},
		{"empty ack", types.NewMsgAcknowledgement(packet, nil, suite.proof, height, addr), false},
		{"missing signer address", types.NewMsgAcknowledgement(packet, packet.GetData(), suite.proof, height, emptyAddr), false},
		{"cannot submit an empty proof", types.NewMsgAcknowledgement(packet, packet.GetData(), emptyProof, height, addr), false},
		{"invalid packet", types.NewMsgAcknowledgement(invalidPacket, packet.GetData(), suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgAcknowledgementGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgAcknowledgement(packet, packet.GetData(), suite.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeInitValidateBasic() {
	var msg *types.MsgChannelUpgradeInit

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			false,
		},
		{
			"empty proposed upgrade channel version",
			func() {
				msg.Fields.Version = "  "
			},
			false,
		},
		{
			"missing signer address",
			func() {
				msg.Signer = emptyAddr
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg = types.NewMsgChannelUpgradeInit(
				ibctesting.MockPort, ibctesting.FirstChannelID,
				types.NewUpgradeFields(types.UNORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
				addr,
			)

			tc.malleate()
			err := msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeInitGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelUpgradeInit(
		ibctesting.MockPort, ibctesting.FirstChannelID,
		types.NewUpgradeFields(types.UNORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
		addr,
	)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeTryValidateBasic() {
	var msg *types.MsgChannelUpgradeTry

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			false,
		},
		{
			"counterparty sequence cannot be zero",
			func() {
				msg.CounterpartyUpgradeSequence = 0
			},
			false,
		},
		{
			"invalid connection hops",
			func() {
				msg.ProposedUpgradeConnectionHops = []string{}
			},
			false,
		},
		{
			"invalid counterparty upgrade fields ordering",
			func() {
				msg.CounterpartyUpgradeFields.Ordering = types.NONE
			},
			false,
		},
		{
			"cannot submit an empty channel proof",
			func() {
				msg.ProofChannel = emptyProof
			},
			false,
		},
		{
			"cannot submit an empty upgrade proof",
			func() {
				msg.ProofUpgrade = emptyProof
			},
			false,
		},
		{
			"missing signer address",
			func() {
				msg.Signer = emptyAddr
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg = types.NewMsgChannelUpgradeTry(
				ibctesting.MockPort,
				ibctesting.FirstChannelID,
				[]string{ibctesting.FirstConnectionID},
				types.NewUpgradeFields(types.UNORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
				1,
				suite.proof,
				suite.proof,
				height,
				addr,
			)

			tc.malleate()
			err := msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeTryGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	msg := types.NewMsgChannelUpgradeTry(
		ibctesting.MockPort,
		ibctesting.FirstChannelID,
		[]string{ibctesting.FirstConnectionID},
		types.NewUpgradeFields(types.UNORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
		1,
		suite.proof,
		suite.proof,
		height,
		addr,
	)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeAckValidateBasic() {
	var msg *types.MsgChannelUpgradeAck

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			false,
		},
		{
			"cannot submit an empty channel proof",
			func() {
				msg.ProofChannel = emptyProof
			},
			false,
		},
		{
			"cannot submit an empty upgrade proof",
			func() {
				msg.ProofUpgrade = emptyProof
			},
			false,
		},
		{
			"missing signer address",
			func() {
				msg.Signer = emptyAddr
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			upgrade := types.NewUpgrade(
				types.NewUpgradeFields(types.ORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
				types.NewTimeout(clienttypes.NewHeight(1, 100), 0),
				0,
			)

			msg = types.NewMsgChannelUpgradeAck(
				ibctesting.MockPort, ibctesting.FirstChannelID,
				upgrade, suite.proof, suite.proof,
				height, addr,
			)

			tc.malleate()
			err := msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeAckGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	upgrade := types.NewUpgrade(
		types.NewUpgradeFields(types.ORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
		types.NewTimeout(clienttypes.NewHeight(1, 100), 0),
		0,
	)

	msg := types.NewMsgChannelUpgradeAck(
		ibctesting.MockPort, ibctesting.FirstChannelID,
		upgrade, suite.proof, suite.proof,
		height, addr,
	)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeConfirmValidateBasic() {
	var msg *types.MsgChannelUpgradeConfirm

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"success: counterparty state set to FLUSHCOMPLETE",
			func() {
				msg.CounterpartyChannelState = types.FLUSHCOMPLETE
			},
			true,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			false,
		},
		{
			"invalid counterparty channel state",
			func() {
				msg.CounterpartyChannelState = types.CLOSED
			},
			false,
		},
		{
			"cannot submit an empty channel proof",
			func() {
				msg.ProofChannel = emptyProof
			},
			false,
		},
		{
			"cannot submit an empty upgrade proof",
			func() {
				msg.ProofUpgrade = emptyProof
			},
			false,
		},
		{
			"missing signer address",
			func() {
				msg.Signer = emptyAddr
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			counterpartyUpgrade := types.NewUpgrade(
				types.NewUpgradeFields(types.UNORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
				types.NewTimeout(clienttypes.NewHeight(0, 10000), timeoutTimestamp),
				0,
			)

			msg = types.NewMsgChannelUpgradeConfirm(
				ibctesting.MockPort, ibctesting.FirstChannelID,
				types.FLUSHING, counterpartyUpgrade, suite.proof, suite.proof,
				height, addr,
			)

			tc.malleate()
			err := msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeConfirmGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)

	msg := &types.MsgChannelUpgradeConfirm{Signer: addr}

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeOpenValidateBasic() {
	var msg *types.MsgChannelUpgradeOpen

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: flushcomplete state",
			func() {},
			true,
		},
		{
			"success: open state",
			func() {
				msg.CounterpartyChannelState = types.OPEN
			},
			true,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			false,
		},
		{
			"invalid counterparty channel state",
			func() {
				msg.CounterpartyChannelState = types.CLOSED
			},
			false,
		},
		{
			"cannot submit an empty channel proof",
			func() {
				msg.ProofChannel = emptyProof
			},
			false,
		},
		{
			"missing signer address",
			func() {
				msg.Signer = emptyAddr
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg = types.NewMsgChannelUpgradeOpen(
				ibctesting.MockPort, ibctesting.FirstChannelID,
				types.FLUSHCOMPLETE, suite.proof,
				height, addr,
			)

			tc.malleate()
			err := msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeTimeoutValidateBasic() {
	var msg *types.MsgChannelUpgradeTimeout

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			false,
		},
		{
			"cannot submit an empty proof",
			func() {
				msg.ProofChannel = emptyProof
			},
			false,
		},
		{
			"invalid counterparty channel state",
			func() {
				msg.CounterpartyChannel.State = types.CLOSED
			},
			false,
		},
		{
			"missing signer address",
			func() {
				msg.Signer = emptyAddr
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg = types.NewMsgChannelUpgradeTimeout(
				ibctesting.MockPort, ibctesting.FirstChannelID,
				types.Channel{State: types.OPEN},
				suite.proof,
				height, addr,
			)

			tc.malleate()
			err := msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeTimeoutGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)

	msg := types.NewMsgChannelUpgradeTimeout(
		ibctesting.MockPort, ibctesting.FirstChannelID,
		types.Channel{},
		suite.proof,
		height, addr,
	)
	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeCancelValidateBasic() {
	var msg *types.MsgChannelUpgradeCancel

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			false,
		},
		{
			"can submit an empty proof",
			func() {
				msg.ProofErrorReceipt = emptyProof
			},
			true,
		},
		{
			"missing signer address",
			func() {
				msg.Signer = emptyAddr
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg = types.NewMsgChannelUpgradeCancel(ibctesting.MockPort, ibctesting.FirstChannelID, types.ErrorReceipt{Sequence: 1}, suite.proof, height, addr)

			tc.malleate()
			err := msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelUpgradeCancelGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)

	msg := types.NewMsgChannelUpgradeCancel(ibctesting.MockPort, ibctesting.FirstChannelID, types.ErrorReceipt{Sequence: 1}, suite.proof, height, addr)
	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}

func (suite *TypesTestSuite) TestMsgPruneAcknowledgementsValidateBasic() {
	var msg *types.MsgPruneAcknowledgements

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: zero pruning limit",
			func() {
				msg.Limit = 0
			},
			types.ErrInvalidPruningLimit,
		},
		{
			"invalid port identifier",
			func() {
				msg.PortId = invalidPort
			},
			host.ErrInvalidID,
		},
		{
			"invalid channel identifier",
			func() {
				msg.ChannelId = invalidChannel
			},
			types.ErrInvalidChannelIdentifier,
		},
		{
			"empty signer address",
			func() {
				msg.Signer = emptyAddr
			},
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg = types.NewMsgPruneAcknowledgements(ibctesting.MockPort, ibctesting.FirstChannelID, 1, addr)

			tc.malleate()
			err := msg.ValidateBasic()

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgUpdateParamsValidateBasic() {
	var msg *types.MsgUpdateParams

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"invalid authority",
			func() {
				msg.Authority = "invalid-address"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"invalid params: non zero height",
			func() {
				newHeight := clienttypes.NewHeight(1, 1000)
				msg = types.NewMsgUpdateChannelParams(authtypes.NewModuleAddress(govtypes.ModuleName).String(), types.NewParams(types.NewTimeout(newHeight, uint64(100000))))
			},
			types.ErrInvalidTimeout,
		},
		{
			"invalid params: zero timestamp",
			func() {
				msg = types.NewMsgUpdateChannelParams(authtypes.NewModuleAddress(govtypes.ModuleName).String(), types.NewParams(types.NewTimeout(clienttypes.ZeroHeight(), uint64(0))))
			},
			types.ErrInvalidTimeout,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg = types.NewMsgUpdateChannelParams(authtypes.NewModuleAddress(govtypes.ModuleName).String(), types.NewParams(types.NewTimeout(clienttypes.ZeroHeight(), uint64(100000))))

			tc.malleate()
			err := msg.ValidateBasic()

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgPruneAcknowledgementsGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)

	msg := types.NewMsgPruneAcknowledgements(ibctesting.MockPort, ibctesting.FirstChannelID, 0, addr)
	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	suite.Require().NoError(err)
	suite.Require().Equal(expSigner.Bytes(), signers[0])
}
