package types_test

import (
	"errors"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/store/iavl"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/testing/simapp"
)

const (
	// valid constants used for testing
	portid   = "testportid"
	chanid   = "channel-0"
	cpportid = "testcpport"
	cpchanid = "testcpchannel"

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

func (s *TypesTestSuite) SetupTest() {
	app := simapp.Setup(s.T(), false)
	db := dbm.NewMemDB()
	store := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	storeKey := storetypes.NewKVStoreKey("iavlStoreKey")

	store.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
	err := store.LoadVersion(0)
	s.Require().NoError(err)
	iavlStore, ok := store.GetCommitStore(storeKey).(*iavl.Store)
	s.Require().True(ok)

	iavlStore.Set([]byte("KEY"), []byte("VALUE"))
	_ = store.Commit()

	res, err := store.Query(&storetypes.RequestQuery{
		Data:  []byte("KEY"),
		Path:  fmt.Sprintf("/%s/key", storeKey.Name()), // required path to get key/value+proof
		Prove: true,
	})
	s.Require().NoError(err)

	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	s.Require().NoError(err)
	proof, err := app.AppCodec().Marshal(&merkleProof)
	s.Require().NoError(err)

	s.proof = proof
}

func TestTypesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

func (s *TypesTestSuite) TestMsgChannelOpenInitValidateBasic() {
	counterparty := types.NewCounterparty(cpportid, cpchanid)
	tryOpenChannel := types.NewChannel(types.TRYOPEN, types.ORDERED, counterparty, connHops, version)

	testCases := []struct {
		name   string
		msg    *types.MsgChannelOpenInit
		expErr error
	}{
		{
			"success",
			types.NewMsgChannelOpenInit(portid, version, types.ORDERED, connHops, cpportid, addr),
			nil,
		},
		{
			"success: empty version",
			types.NewMsgChannelOpenInit(portid, "", types.UNORDERED, connHops, cpportid, addr),
			nil,
		},
		{
			"too short port id",
			types.NewMsgChannelOpenInit(invalidShortPort, version, types.ORDERED, connHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortPort,
					len(invalidShortPort),
					2,
					host.DefaultMaxPortCharacterLength,
				), "invalid port ID"),
		},
		{
			"too long port id",
			types.NewMsgChannelOpenInit(invalidLongPort, version, types.ORDERED, connHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongPort,
					len(invalidLongPort),
					2,
					host.DefaultMaxPortCharacterLength,
				), "invalid port ID"),
		},
		{
			"port id contains non-alpha",
			types.NewMsgChannelOpenInit(invalidPort, version, types.ORDERED, connHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid port ID"),
		},
		{
			"invalid channel order",
			types.NewMsgChannelOpenInit(portid, version, types.Order(3),
				connHops, cpportid, addr),
			errorsmod.Wrap(types.ErrInvalidChannelOrdering, types.Order(3).String()),
		},
		{
			"connection hops more than 1 ",
			types.NewMsgChannelOpenInit(portid, version, types.ORDERED, invalidConnHops, cpportid, addr),
			errorsmod.Wrap(
				types.ErrTooManyConnectionHops,
				"current IBC version only supports one connection hop",
			),
		},
		{
			"too short connection id",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidShortConnHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortConnection, len(invalidShortConnection), 10, host.DefaultMaxCharacterLength),
				"invalid connection hop ID",
			),
		},
		{
			"too long connection id",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidLongConnHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongConnection, len(invalidLongConnection), 10, host.DefaultMaxCharacterLength),
				"invalid connection hop ID",
			),
		},
		{
			"connection id contains non-alpha",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, []string{invalidConnection}, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidConnection,
				), "invalid connection hop ID",
			),
		},
		{
			"invalid counterparty port id",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, connHops, invalidPort, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid counterparty port ID",
			),
		},
		{
			"channel not in INIT state",
			&types.MsgChannelOpenInit{portid, tryOpenChannel, addr},
			errorsmod.Wrapf(types.ErrInvalidChannelState,
				"channel state must be INIT in MsgChannelOpenInit. expected: %s, got: %s",
				types.INIT, tryOpenChannel.State,
			),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgChannelOpenInitGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgChannelOpenInit(portid, version, types.ORDERED, connHops, cpportid, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgChannelOpenTryValidateBasic() {
	counterparty := types.NewCounterparty(cpportid, cpchanid)
	initChannel := types.NewChannel(types.INIT, types.ORDERED, counterparty, connHops, version)

	testCases := []struct {
		name   string
		msg    *types.MsgChannelOpenTry
		expErr error
	}{
		{
			"success",
			types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, version, s.proof, height, addr),
			nil,
		},
		{
			"success with empty channel version",
			types.NewMsgChannelOpenTry(portid, "", types.UNORDERED, connHops, cpportid, cpchanid, version, s.proof, height, addr),
			nil,
		},
		{
			"success with empty counterparty version",
			types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, "", s.proof, height, addr),
			nil,
		},
		{
			"too short port id",
			types.NewMsgChannelOpenTry(invalidShortPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortPort, len(invalidShortPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"too long port id",
			types.NewMsgChannelOpenTry(invalidLongPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongPort, len(invalidLongPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"port id contains non-alpha",
			types.NewMsgChannelOpenTry(invalidPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid port ID",
			),
		},
		{
			"invalid channel order",
			types.NewMsgChannelOpenTry(portid, version, types.Order(4), connHops, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(types.ErrInvalidChannelOrdering, types.Order(4).String()),
		},
		{
			"connection hops more than 1 ",
			types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidConnHops, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				types.ErrTooManyConnectionHops,
				"current IBC version only supports one connection hop",
			),
		},
		{
			"too short connection id",
			types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidShortConnHops, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters", invalidShortConnection, len(invalidShortConnection), 10, host.DefaultMaxCharacterLength),
				"invalid connection hop ID",
			),
		},
		{
			"too long connection id",
			types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidLongConnHops, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters", invalidLongConnection, len(invalidLongConnection), 10, host.DefaultMaxCharacterLength),
				"invalid connection hop ID",
			),
		},
		{
			"connection id contains non-alpha",
			types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, []string{invalidConnection}, cpportid, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidConnection,
				), "invalid connection hop ID",
			),
		},
		{
			"invalid counterparty port id",
			types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, invalidPort, cpchanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid counterparty port ID",
			),
		},
		{
			"invalid counterparty channel id",
			types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, cpportid, invalidChannel, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidChannel,
				), "invalid counterparty channel ID",
			),
		},
		{
			"empty proof",
			types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, cpportid, cpchanid, version, emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty init proof"),
		},
		{
			"channel not in TRYOPEN state",
			&types.MsgChannelOpenTry{portid, "", initChannel, version, s.proof, height, addr},
			errorsmod.Wrapf(types.ErrInvalidChannelState,
				"channel state must be TRYOPEN in MsgChannelOpenTry. expected: %s, got: %s",
				types.TRYOPEN, initChannel.State,
			),
		},
		{
			"previous channel id is not empty",
			&types.MsgChannelOpenTry{portid, chanid, initChannel, version, s.proof, height, addr},
			errorsmod.Wrap(types.ErrInvalidChannelIdentifier, "previous channel identifier must be empty, this field has been deprecated as crossing hellos are no longer supported"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgChannelOpenTryGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, version, s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgChannelOpenAckValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgChannelOpenAck
		expErr error
	}{
		{
			"success",
			types.NewMsgChannelOpenAck(portid, chanid, chanid, version, s.proof, height, addr),
			nil,
		},
		{
			"success empty cpv",
			types.NewMsgChannelOpenAck(portid, chanid, chanid, "", s.proof, height, addr),
			nil,
		},
		{
			"too short port id",
			types.NewMsgChannelOpenAck(invalidShortPort, chanid, chanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortPort, len(invalidShortPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"too long port id",
			types.NewMsgChannelOpenAck(invalidLongPort, chanid, chanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongPort, len(invalidLongPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"port id contains non-alpha",
			types.NewMsgChannelOpenAck(invalidPort, chanid, chanid, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid port ID",
			),
		},
		{
			"too short channel id",
			types.NewMsgChannelOpenAck(portid, invalidShortChannel, chanid, version, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"too long channel id",
			types.NewMsgChannelOpenAck(portid, invalidLongChannel, chanid, version, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"channel id contains non-alpha",
			types.NewMsgChannelOpenAck(portid, invalidChannel, chanid, version, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"empty proof",
			types.NewMsgChannelOpenAck(portid, chanid, chanid, version, emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty try proof"),
		},
		{
			"invalid counterparty channel id",
			types.NewMsgChannelOpenAck(portid, chanid, invalidShortChannel, version, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortChannel, len(invalidShortChannel), 8, host.DefaultMaxCharacterLength),
				"invalid counterparty channel ID",
			),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgChannelOpenAckGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgChannelOpenAck(portid, chanid, chanid, version, s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgChannelOpenConfirmValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgChannelOpenConfirm
		expErr error
	}{
		{
			"success",
			types.NewMsgChannelOpenConfirm(portid, chanid, s.proof, height, addr),
			nil,
		},
		{
			"too short port id",
			types.NewMsgChannelOpenConfirm(invalidShortPort, chanid, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortPort, len(invalidShortPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"too long port id",
			types.NewMsgChannelOpenConfirm(invalidLongPort, chanid, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongPort, len(invalidLongPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"port id contains non-alpha",
			types.NewMsgChannelOpenConfirm(invalidPort, chanid, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid port ID",
			),
		},
		{
			"too short channel id",
			types.NewMsgChannelOpenConfirm(portid, invalidShortChannel, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"too long channel id",
			types.NewMsgChannelOpenConfirm(portid, invalidLongChannel, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"channel id contains non-alpha",
			types.NewMsgChannelOpenConfirm(portid, invalidChannel, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"empty proof",
			types.NewMsgChannelOpenConfirm(portid, chanid, emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty acknowledgement proof"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgChannelOpenConfirmGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgChannelOpenConfirm(portid, chanid, s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgChannelCloseInitValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgChannelCloseInit
		expErr error
	}{
		{
			"success",
			types.NewMsgChannelCloseInit(portid, chanid, addr),
			nil,
		},
		{
			"too short port id",
			types.NewMsgChannelCloseInit(invalidShortPort, chanid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortPort, len(invalidShortPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"too long port id",
			types.NewMsgChannelCloseInit(invalidLongPort, chanid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongPort, len(invalidLongPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"port id contains non-alpha",
			types.NewMsgChannelCloseInit(invalidPort, chanid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid port ID",
			),
		},
		{
			"too short channel id",
			types.NewMsgChannelCloseInit(portid, invalidShortChannel, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"too long channel id",
			types.NewMsgChannelCloseInit(portid, invalidLongChannel, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"channel id contains non-alpha",
			types.NewMsgChannelCloseInit(portid, invalidChannel, addr),
			types.ErrInvalidChannelIdentifier,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgChannelCloseInitGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgChannelCloseInit(portid, chanid, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgChannelCloseConfirmValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgChannelCloseConfirm
		expErr error
	}{
		{
			"success",
			types.NewMsgChannelCloseConfirm(portid, chanid, s.proof, height, addr),
			nil,
		},
		{
			"success, positive counterparty upgrade sequence",
			types.NewMsgChannelCloseConfirm(portid, chanid, s.proof, height, addr),
			nil,
		},
		{
			"too short port id",
			types.NewMsgChannelCloseConfirm(invalidShortPort, chanid, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortPort, len(invalidShortPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"too long port id",
			types.NewMsgChannelCloseConfirm(invalidLongPort, chanid, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongPort, len(invalidLongPort), 2, host.DefaultMaxPortCharacterLength),
				"invalid port ID",
			),
		},
		{
			"port id contains non-alpha",
			types.NewMsgChannelCloseConfirm(invalidPort, chanid, s.proof, height, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid port ID",
			),
		},
		{
			"too short channel id",
			types.NewMsgChannelCloseConfirm(portid, invalidShortChannel, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"too long channel id",
			types.NewMsgChannelCloseConfirm(portid, invalidLongChannel, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"channel id contains non-alpha",
			types.NewMsgChannelCloseConfirm(portid, invalidChannel, s.proof, height, addr),
			types.ErrInvalidChannelIdentifier,
		},
		{
			"empty proof",
			types.NewMsgChannelCloseConfirm(portid, chanid, emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty init proof"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgChannelCloseConfirmGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgChannelCloseConfirm(portid, chanid, s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgRecvPacketValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgRecvPacket
		expErr error
	}{
		{
			"success",
			types.NewMsgRecvPacket(packet, s.proof, height, addr),
			nil,
		},
		{
			"missing signer address",
			types.NewMsgRecvPacket(packet, s.proof, height, emptyAddr),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", errors.New("empty address string is not allowed")),
		},
		{
			"proof contain empty proof",
			types.NewMsgRecvPacket(packet, emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty commitment proof"),
		},
		{
			"invalid packet",
			types.NewMsgRecvPacket(invalidPacket, s.proof, height, addr),
			errorsmod.Wrap(types.ErrInvalidPacket, "packet sequence cannot be 0"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgRecvPacketGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgRecvPacket(packet, s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgTimeoutValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgTimeout
		expErr error
	}{
		{
			"success",
			types.NewMsgTimeout(packet, 1, s.proof, height, addr),
			nil,
		},
		{
			"seq 0",
			types.NewMsgTimeout(packet, 0, s.proof, height, addr),
			errorsmod.Wrap(ibcerrors.ErrInvalidSequence, "next sequence receive cannot be 0"),
		},
		{
			"missing signer address",
			types.NewMsgTimeout(packet, 1, s.proof, height, emptyAddr),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", errors.New("empty address string is not allowed")),
		},
		{
			"cannot submit an empty proof",
			types.NewMsgTimeout(packet, 1, emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty unreceived proof"),
		},
		{
			"invalid packet",
			types.NewMsgTimeout(invalidPacket, 1, s.proof, height, addr),
			errorsmod.Wrap(types.ErrInvalidPacket, "packet sequence cannot be 0"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgTimeoutGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgTimeout(packet, 1, s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgTimeoutOnCloseValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgTimeoutOnClose
		expErr error
	}{
		{
			"success",
			types.NewMsgTimeoutOnClose(packet, 1, s.proof, s.proof, height, addr),
			nil,
		},
		{
			"success, positive counterparty upgrade sequence",
			types.NewMsgTimeoutOnClose(packet, 1, s.proof, s.proof, height, addr),
			nil,
		},
		{
			"seq 0",
			types.NewMsgTimeoutOnClose(packet, 0, s.proof, s.proof, height, addr),
			errorsmod.Wrap(ibcerrors.ErrInvalidSequence, "next sequence receive cannot be 0"),
		},
		{
			"signer address is empty",
			types.NewMsgTimeoutOnClose(packet, 1, s.proof, s.proof, height, emptyAddr),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", errors.New("empty address string is not allowed")),
		},
		{
			"empty proof",
			types.NewMsgTimeoutOnClose(packet, 1, emptyProof, s.proof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty unreceived proof"),
		},
		{
			"empty proof close",
			types.NewMsgTimeoutOnClose(packet, 1, s.proof, emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty proof of closed counterparty channel end"),
		},
		{
			"invalid packet",
			types.NewMsgTimeoutOnClose(invalidPacket, 1, s.proof, s.proof, height, addr),
			errorsmod.Wrap(types.ErrInvalidPacket, "packet sequence cannot be 0"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgTimeoutOnCloseGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgTimeoutOnClose(packet, 1, s.proof, s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}

func (s *TypesTestSuite) TestMsgAcknowledgementValidateBasic() {
	testCases := []struct {
		name   string
		msg    *types.MsgAcknowledgement
		expErr error
	}{
		{
			"success",
			types.NewMsgAcknowledgement(packet, packet.GetData(), s.proof, height, addr),
			nil,
		},
		{
			"empty ack",
			types.NewMsgAcknowledgement(packet, nil, s.proof, height, addr),
			errorsmod.Wrap(types.ErrInvalidAcknowledgement, "ack bytes cannot be empty"),
		},
		{
			"missing signer address",
			types.NewMsgAcknowledgement(packet, packet.GetData(), s.proof, height, emptyAddr),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", errors.New("empty address string is not allowed")),
		},
		{
			"cannot submit an empty proof",
			types.NewMsgAcknowledgement(packet, packet.GetData(), emptyProof, height, addr),
			errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty acknowledgement proof"),
		},
		{
			"invalid packet",
			types.NewMsgAcknowledgement(invalidPacket, packet.GetData(), s.proof, height, addr),
			errorsmod.Wrap(types.ErrInvalidPacket, "packet sequence cannot be 0"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgAcknowledgementGetSigners() {
	expSigner, err := sdk.AccAddressFromBech32(addr)
	s.Require().NoError(err)
	msg := types.NewMsgAcknowledgement(packet, packet.GetData(), s.proof, height, addr)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)

	s.Require().NoError(err)
	s.Require().Equal(expSigner.Bytes(), signers[0])
}
