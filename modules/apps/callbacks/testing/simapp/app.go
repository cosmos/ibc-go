package simapp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cast"

	coreaddress "cosmossdk.io/core/address"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/accounts"
	"cosmossdk.io/x/accounts/accountstd"
	baseaccount "cosmossdk.io/x/accounts/defaults/base"
	"cosmossdk.io/x/accounts/defaults/lockup"
	"cosmossdk.io/x/accounts/defaults/multisig"
	"cosmossdk.io/x/bank"
	bankkeeper "cosmossdk.io/x/bank/keeper"
	banktypes "cosmossdk.io/x/bank/types"
	"cosmossdk.io/x/consensus"
	consensusparamkeeper "cosmossdk.io/x/consensus/keeper"
	consensusparamtypes "cosmossdk.io/x/consensus/types"
	distr "cosmossdk.io/x/distribution"
	distrkeeper "cosmossdk.io/x/distribution/keeper"
	distrtypes "cosmossdk.io/x/distribution/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/gov"
	govkeeper "cosmossdk.io/x/gov/keeper"
	govtypes "cosmossdk.io/x/gov/types"
	"cosmossdk.io/x/mint"
	mintkeeper "cosmossdk.io/x/mint/keeper"
	minttypes "cosmossdk.io/x/mint/types"
	"cosmossdk.io/x/params"
	paramskeeper "cosmossdk.io/x/params/keeper"
	paramstypes "cosmossdk.io/x/params/types"
	poolkeeper "cosmossdk.io/x/protocolpool/keeper"
	pooltypes "cosmossdk.io/x/protocolpool/types"
	"cosmossdk.io/x/slashing"
	slashingkeeper "cosmossdk.io/x/slashing/keeper"
	slashingtypes "cosmossdk.io/x/slashing/types"
	"cosmossdk.io/x/staking"
	stakingkeeper "cosmossdk.io/x/staking/keeper"
	stakingtypes "cosmossdk.io/x/staking/types"
	txdecode "cosmossdk.io/x/tx/decode"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil/testdata/testpb"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/crypto"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"

	ibccallbacks "github.com/cosmos/ibc-go/modules/apps/callbacks"
	ica "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v9/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v9/modules/core"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
	ibctestingtypes "github.com/cosmos/ibc-go/v9/testing/types"
)

const appName = "SimApp"

// IBC application testing ports
const (
	MockFeePort string = ibcmock.ModuleName + ibcfeetypes.ModuleName
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:         nil,
		distrtypes.ModuleName:              nil,
		pooltypes.ModuleName:               nil,
		pooltypes.StreamAccount:            nil,
		pooltypes.ProtocolPoolDistrAccount: nil,
		minttypes.ModuleName:               {authtypes.Minter},
		stakingtypes.BondedPoolName:        {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:     {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:                {authtypes.Burner},
		ibctransfertypes.ModuleName:        {authtypes.Minter, authtypes.Burner},
		ibcfeetypes.ModuleName:             nil,
		icatypes.ModuleName:                nil,
		ibcmock.ModuleName:                 nil,
	}
)

var _ servertypes.Application = (*SimApp)(nil)

// SimApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions.
type SimApp struct {
	*baseapp.BaseApp
	logger            log.Logger
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountsKeeper accounts.Keeper
	AuthKeeper     authkeeper.AccountKeeper
	BankKeeper     bankkeeper.BaseKeeper
	FeeGrantKeeper feegrantkeeper.Keeper
	StakingKeeper  *stakingkeeper.Keeper
	SlashingKeeper slashingkeeper.Keeper
	MintKeeper     mintkeeper.Keeper
	DistrKeeper    distrkeeper.Keeper
	PoolKeeper     poolkeeper.Keeper
	GovKeeper      govkeeper.Keeper
	UpgradeKeeper  *upgradekeeper.Keeper
	ParamsKeeper   paramskeeper.Keeper

	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCFeeKeeper          ibcfeekeeper.Keeper
	ICAControllerKeeper   icacontrollerkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// mock contract keeper used for testing
	MockContractKeeper *ContractKeeper

	// make IBC modules public for test purposes
	// these modules are never directly routed to by the IBC Router
	ICAAuthModule ibcmock.IBCModule
	FeeMockModule ibcmock.IBCModule

	// the module manager
	ModuleManager *module.Manager

	// simulation manager
	simulationManager *module.SimulationManager

	// module configurator
	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".simapp")
}

// NewSimApp returns a reference to an initialized SimApp.
func NewSimApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *SimApp {
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
			ValidatorAddressCodec: address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	signingCtx := interfaceRegistry.SigningContext()
	txDecoder, err := txdecode.NewDecoder(txdecode.Options{
		SigningContext: signingCtx,
		ProtoCodec:     appCodec,
	})
	if err != nil {
		panic(err)
	}
	txConfig := authtx.NewTxConfig(appCodec, signingCtx.AddressCodec(), signingCtx.ValidatorAddressCodec(), authtx.DefaultSignModes)

	govModuleAddr, err := signingCtx.AddressCodec().BytesToString(authtypes.NewModuleAddress(govtypes.ModuleName))
	if err != nil {
		panic(err)
	}

	if err := signingCtx.Validate(); err != nil {
		panic(err)
	}

	std.RegisterLegacyAminoCodec(legacyAmino)
	std.RegisterInterfaces(interfaceRegistry)

	// Below we could construct and set an application specific mempool and
	// ABCI 1.0 PrepareProposal and ProcessProposal handlers. These defaults are
	// already set in the SDK's BaseApp, this shows an example of how to override
	// them.
	//
	// Example:
	//
	// bApp := baseapp.NewBaseApp(...)
	// nonceMempool := mempool.NewSenderNonceMempool()
	// abciPropHandler := NewDefaultProposalHandler(nonceMempool, bApp)
	//
	// bApp.SetMempool(nonceMempool)
	// bApp.SetPrepareProposal(abciPropHandler.PrepareProposalHandler())
	// bApp.SetProcessProposal(abciPropHandler.ProcessProposalHandler())
	//
	// Alternatively, you can construct BaseApp options, append those to
	// baseAppOptions and pass them to NewBaseApp.
	//
	// Example:
	//
	// prepareOpt = func(app *baseapp.BaseApp) {
	// 	abciPropHandler := baseapp.NewDefaultProposalHandler(nonceMempool, app)
	// 	app.SetPrepareProposal(abciPropHandler.PrepareProposalHandler())
	// }
	// baseAppOptions = append(baseAppOptions, prepareOpt)

	bApp := baseapp.NewBaseApp(appName, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		accounts.StoreKey, authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		minttypes.StoreKey, distrtypes.StoreKey, pooltypes.StoreKey, slashingtypes.StoreKey,
		feegrant.StoreKey, govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey, upgradetypes.StoreKey,
		ibctransfertypes.StoreKey, icacontrollertypes.StoreKey, icahosttypes.StoreKey,
		ibcfeetypes.StoreKey, consensusparamtypes.StoreKey,
	)

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, keys); err != nil {
		panic(err)
	}

	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := storetypes.NewMemoryStoreKeys(ibcmock.MemStoreKey)

	app := &SimApp{
		BaseApp:           bApp,
		logger:            logger,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}
	cometService := runtime.NewContextAwareCometInfoService()

	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, runtime.NewEnvironment(runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]), logger.With(log.ModuleKey, "x/consensus")), govModuleAddr)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// SDK module keepers

	// add keepers
	accountsKeeper, err := accounts.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[accounts.StoreKey]), logger.With(log.ModuleKey, "x/accounts"), runtime.EnvWithMsgRouterService(app.MsgServiceRouter()), runtime.EnvWithQueryRouterService(app.GRPCQueryRouter())),
		signingCtx.AddressCodec(),
		appCodec.InterfaceRegistry(),
		txDecoder,
		// Lockup account
		accountstd.AddAccount(lockup.CONTINUOUS_LOCKING_ACCOUNT, lockup.NewContinuousLockingAccount),
		accountstd.AddAccount(lockup.PERIODIC_LOCKING_ACCOUNT, lockup.NewPeriodicLockingAccount),
		accountstd.AddAccount(lockup.DELAYED_LOCKING_ACCOUNT, lockup.NewDelayedLockingAccount),
		accountstd.AddAccount(lockup.PERMANENT_LOCKING_ACCOUNT, lockup.NewPermanentLockingAccount),
		accountstd.AddAccount("multisig", multisig.NewAccount),
		// PRODUCTION: add
		baseaccount.NewAccount("base", txConfig.SignModeHandler(), baseaccount.WithSecp256K1PubKey()),
	)
	if err != nil {
		panic(err)
	}

	app.AccountsKeeper = accountsKeeper

	app.AuthKeeper = authkeeper.NewAccountKeeper(runtime.NewEnvironment(runtime.NewKVStoreService(keys[authtypes.StoreKey]), logger.With(log.ModuleKey, "x/auth")), appCodec, authtypes.ProtoBaseAccount, accountsKeeper, maccPerms, signingCtx.AddressCodec(), sdk.Bech32MainPrefix, govModuleAddr)

	blockedAddrs, err := BlockedAddresses(signingCtx.AddressCodec())
	if err != nil {
		panic(err)
	}

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[banktypes.StoreKey]), logger.With(log.ModuleKey, "x/bank")),
		appCodec,
		app.AuthKeeper,
		blockedAddrs,
		govModuleAddr,
	)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(
			runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
			logger.With(log.ModuleKey, "x/staking"),
			runtime.EnvWithMsgRouterService(app.MsgServiceRouter()),
			runtime.EnvWithQueryRouterService(app.GRPCQueryRouter())),
		app.AuthKeeper,
		app.BankKeeper,
		app.ConsensusParamsKeeper,
		govModuleAddr,
		signingCtx.ValidatorAddressCodec(),
		authcodec.NewBech32Codec(sdk.Bech32PrefixConsAddr),
		cometService,
	)

	app.MintKeeper = mintkeeper.NewKeeper(appCodec, runtime.NewEnvironment(runtime.NewKVStoreService(keys[minttypes.StoreKey]), logger.With(log.ModuleKey, "x/mint")), app.StakingKeeper, app.AuthKeeper, app.BankKeeper, authtypes.FeeCollectorName, govModuleAddr)

	app.PoolKeeper = poolkeeper.NewKeeper(appCodec, runtime.NewEnvironment(runtime.NewKVStoreService(keys[pooltypes.StoreKey]), logger.With(log.ModuleKey, "x/protocolpool")), app.AuthKeeper, app.BankKeeper, app.StakingKeeper, govModuleAddr)

	app.DistrKeeper = distrkeeper.NewKeeper(appCodec, runtime.NewEnvironment(runtime.NewKVStoreService(keys[distrtypes.StoreKey]), logger.With(log.ModuleKey, "x/distribution")), app.AuthKeeper, app.BankKeeper, app.StakingKeeper, cometService, authtypes.FeeCollectorName, govModuleAddr)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(runtime.NewEnvironment(runtime.NewKVStoreService(keys[feegrant.StoreKey]), logger.With(log.ModuleKey, "x/feegrant")), appCodec, app.AuthKeeper)

	app.SlashingKeeper = slashingkeeper.NewKeeper(runtime.NewEnvironment(runtime.NewKVStoreService(keys[slashingtypes.StoreKey]), logger.With(log.ModuleKey, "x/slashing")),
		appCodec, legacyAmino, app.StakingKeeper, govModuleAddr,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	// set the governance module account as the authority for conducting upgrades
	app.UpgradeKeeper = upgradekeeper.NewKeeper(runtime.NewEnvironment(runtime.NewKVStoreService(keys[upgradetypes.StoreKey]), logger.With(log.ModuleKey, "x/upgrade"), runtime.EnvWithMsgRouterService(app.MsgServiceRouter()), runtime.EnvWithQueryRouterService(app.GRPCQueryRouter())), skipUpgradeHeights, appCodec, homePath, app.BaseApp, govModuleAddr, app.ConsensusParamsKeeper)

	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[ibcexported.StoreKey]), app.GetSubspace(ibcexported.ModuleName), app.UpgradeKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// NOTE: The mock ContractKeeper is only created for testing.
	// Real applications should not use the mock ContractKeeper
	app.MockContractKeeper = NewContractKeeper(memKeys[ibcmock.MemStoreKey])

	govConfig := govkeeper.DefaultConfig()
	/*
		Example of setting gov params:
		govConfig.MaxMetadataLen = 10000
	*/
	govKeeper := govkeeper.NewKeeper(appCodec, runtime.NewEnvironment(runtime.NewKVStoreService(keys[govtypes.StoreKey]), logger.With(log.ModuleKey, "x/gov"), runtime.EnvWithMsgRouterService(app.MsgServiceRouter()), runtime.EnvWithQueryRouterService(app.GRPCQueryRouter())), app.AuthKeeper, app.BankKeeper, app.StakingKeeper, app.PoolKeeper, govConfig, govModuleAddr)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(
		// register the governance hooks
		),
	)

	// IBC Fee Module keeper
	app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[ibcfeetypes.StoreKey]),
		app.IBCKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.IBCKeeper.ChannelKeeper,
		app.AuthKeeper, app.BankKeeper,
	)

	// ICA Controller keeper
	app.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[icacontrollertypes.StoreKey]), logger.With(log.ModuleKey, "x/icacontroller"), runtime.EnvWithMsgRouterService(app.MsgServiceRouter())),
		app.GetSubspace(icacontrollertypes.SubModuleName),
		app.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		app.IBCKeeper.ChannelKeeper,
		govModuleAddr,
	)

	// ICA Host keeper
	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[icahosttypes.StoreKey]), logger.With(log.ModuleKey, "x/icahost"), runtime.EnvWithMsgRouterService(app.MsgServiceRouter()), runtime.EnvWithQueryRouterService(app.GRPCQueryRouter())),
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		app.IBCKeeper.ChannelKeeper,
		app.AuthKeeper,
		govModuleAddr,
	)

	// Create IBC Router
	ibcRouter := porttypes.NewRouter()

	// Middleware Stacks
	maxCallbackGas := uint64(1_000_000)

	// Create Transfer Keeper and pass IBCFeeKeeper as expected Channel and PortKeeper
	// since fee middleware will wrap the IBCKeeper for underlying application.
	// NOTE: the Transfer Keeper's ICS4Wrapper can later be replaced.
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[ibctransfertypes.StoreKey]), app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCFeeKeeper, // ISC4 Wrapper: fee IBC middleware
		app.IBCKeeper.ChannelKeeper,
		app.AuthKeeper, app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Mock Module Stack

	// Mock Module setup for testing IBC and also acts as the interchain accounts authentication module
	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	mockModule := ibcmock.NewAppModule()

	// The mock module is used for testing IBC
	mockIBCModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(ibcmock.ModuleName))
	ibcRouter.AddRoute(ibcmock.ModuleName, mockIBCModule)

	// Create Transfer Stack
	// SendPacket, since it is originating from the application to core IBC:
	// transferKeeper.SendPacket -> callbacks.SendPacket -> feeKeeper.SendPacket -> channel.SendPacket

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
	// channel.RecvPacket -> fee.OnRecvPacket -> callbacks.OnRecvPacket -> transfer.OnRecvPacket

	// transfer stack contains (from top to bottom):
	// - IBC Fee Middleware
	// - IBC Callbacks Middleware
	// - Transfer

	// create IBC module from bottom to top of stack
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = ibccallbacks.NewIBCMiddleware(transferStack, app.IBCFeeKeeper, app.MockContractKeeper, maxCallbackGas)
	var transferICS4Wrapper porttypes.ICS4Wrapper
	transferICS4Wrapper, ok := transferStack.(porttypes.ICS4Wrapper)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", transferStack, transferICS4Wrapper))
	}

	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)
	// Since the callbacks middleware itself is an ics4wrapper, it needs to be passed to the transfer keeper
	app.TransferKeeper.WithICS4Wrapper(transferICS4Wrapper)

	// Add transfer stack to IBC Router
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)

	// Create Interchain Accounts Stack
	// SendPacket, since it is originating from the application to core IBC:
	// icaControllerKeeper.SendTx -> callbacks.SendPacket -> fee.SendPacket -> channel.SendPacket

	// initialize ICA module with mock module as the authentication module on the controller side
	var icaControllerStack porttypes.IBCModule
	icaControllerStack = ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(""))
	app.ICAAuthModule, ok = icaControllerStack.(ibcmock.IBCModule)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", icaControllerStack, app.ICAAuthModule))
	}
	icaControllerStack = icacontroller.NewIBCMiddlewareWithAuth(icaControllerStack, app.ICAControllerKeeper)
	icaControllerStack = ibccallbacks.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper, app.MockContractKeeper, maxCallbackGas)
	var icaICS4Wrapper porttypes.ICS4Wrapper
	icaICS4Wrapper, ok = icaControllerStack.(porttypes.ICS4Wrapper)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", icaControllerStack, icaICS4Wrapper))
	}
	icaControllerStack = ibcfee.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper)
	// Since the callbacks middleware itself is an ics4wrapper, it needs to be passed to the ica controller keeper
	app.ICAControllerKeeper.WithICS4Wrapper(icaICS4Wrapper)

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket

	var icaHostStack porttypes.IBCModule
	icaHostStack = icahost.NewIBCModule(app.ICAHostKeeper)
	icaHostStack = ibcfee.NewIBCMiddleware(icaHostStack, app.IBCFeeKeeper)

	// Add host, controller & ica auth modules to IBC router
	ibcRouter.
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(ibcmock.ModuleName+icacontrollertypes.SubModuleName, icaControllerStack) // ica with mock auth module stack route to ica (top level of middleware stack)

	// Create Mock IBC Fee module stack for testing
	// SendPacket, mock module cannot send packets

	// OnRecvPacket, message that originates from core IBC and goes down to app, the flow is the otherway
	// channel.RecvPacket -> callbacks.OnRecvPacket -> fee.OnRecvPacket -> mockModule.OnRecvPacket

	// OnAcknowledgementPacket as this is where fee's are paid out
	// mockModule.OnAcknowledgementPacket -> callbacks.OnAcknowledgementPacket -> fee.OnAcknowledgementPacket -> channel.OnAcknowledgementPacket

	// create fee wrapped mock module
	feeMockModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(MockFeePort))
	app.FeeMockModule = feeMockModule
	var feeWithMockModule porttypes.Middleware = ibcfee.NewIBCMiddleware(feeMockModule, app.IBCFeeKeeper)
	feeWithMockModule = ibccallbacks.NewIBCMiddleware(feeWithMockModule, app.IBCFeeKeeper, app.MockContractKeeper, maxCallbackGas)
	ibcRouter.AddRoute(MockFeePort, feeWithMockModule)

	// Seal the IBC Router
	app.IBCKeeper.SetRouter(ibcRouter)

	clientKeeper := app.IBCKeeper.ClientKeeper
	storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()

	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

	smLightClientModule := solomachine.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(solomachine.ModuleName, &smLightClientModule)

	// ****  Module Options ****

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.ModuleManager = module.NewManager(
		genutil.NewAppModule(appCodec, app.AuthKeeper, app.StakingKeeper, app, txConfig, genutiltypes.DefaultMessageValidator),
		accounts.NewAppModule(appCodec, app.AccountsKeeper),
		auth.NewAppModule(appCodec, app.AuthKeeper, app.AccountsKeeper, authsims.RandomGenesisAccounts, nil),
		vesting.NewAppModule(app.AuthKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AuthKeeper),
		feegrantmodule.NewAppModule(appCodec, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AuthKeeper, app.BankKeeper, app.PoolKeeper),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AuthKeeper, nil),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AuthKeeper, app.BankKeeper, app.StakingKeeper, app.interfaceRegistry, cometService),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.StakingKeeper),
		staking.NewAppModule(appCodec, app.StakingKeeper),
		upgrade.NewAppModule(app.UpgradeKeeper),
		params.NewAppModule(app.ParamsKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),

		// IBC modules
		ibc.NewAppModule(appCodec, app.IBCKeeper),
		transfer.NewAppModule(appCodec, app.TransferKeeper),
		ibcfee.NewAppModule(appCodec, app.IBCFeeKeeper),
		ica.NewAppModule(appCodec, &app.ICAControllerKeeper, &app.ICAHostKeeper),
		mockModule,

		// IBC light clients
		ibctm.NewAppModule(tmLightClientModule),
		solomachine.NewAppModule(smLightClientModule),
	)

	app.ModuleManager.RegisterLegacyAminoCodec(legacyAmino)
	app.ModuleManager.RegisterInterfaces(interfaceRegistry)

	// NOTE: upgrade module is required to be prioritized
	app.ModuleManager.SetOrderPreBlockers(
		upgradetypes.ModuleName,
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.ModuleManager.SetOrderBeginBlockers(
		minttypes.ModuleName,
		distrtypes.ModuleName,
		pooltypes.ModuleName,
		slashingtypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		genutiltypes.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		ibcmock.ModuleName,
	)
	app.ModuleManager.SetOrderEndBlockers(
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		feegrant.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		genutiltypes.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		ibcmock.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
	genesisModuleOrder := []string{
		consensusparamtypes.ModuleName,
		accounts.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		pooltypes.ModuleName,
		feegrant.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		genutiltypes.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		ibcmock.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
	}
	app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	// Uncomment if you want to set a custom migration order here.
	// app.ModuleManager.SetOrderMigrations(custom order)

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err = app.ModuleManager.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	// add test gRPC service for testing gRPC queries in isolation
	testpb.RegisterQueryServer(app.GRPCQueryRouter(), testpb.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AuthKeeper, app.AccountsKeeper, authsims.RandomGenesisAccounts, nil),
	}

	app.simulationManager = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)
	app.simulationManager.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.setAnteHandler(txConfig)

	// In v0.46, the SDK introduces _postHandlers_. PostHandlers are like
	// antehandlers, but are run _after_ the `runMsgs` execution. They are also
	// defined as a chain, and have the same signature as antehandlers.
	//
	// In baseapp, postHandlers are run in the same store branch as `runMsgs`,
	// meaning that both `runMsgs` and `postHandler` state will be committed if
	// both are successful, and both will be reverted if any of the two fails.
	//
	// The SDK exposes a default postHandlers chain, which is comprised of only
	// one decorator: the Transaction Tips decorator. However, some chains do
	// not need it by default, so feel free to comment the next line if you do
	// not need tips.
	// To read more about tips:
	// https://docs.cosmos.network/main/core/tips.html
	//
	// Please note that changing any of the anteHandler or postHandler chain is
	// likely to be a state-machine breaking change, which needs a coordinated
	// upgrade.
	app.setPostHandler()

	// At startup, after all modules have been registered, check that all proto
	// annotations are correct.
	protoFiles, err := proto.MergedRegistry()
	if err != nil {
		panic(err)
	}
	err = msgservice.ValidateProtoAnnotations(protoFiles)
	if err != nil {
		// Once we switch to using protoreflect-based antehandlers, we might
		// want to panic here instead of logging a warning.
		_, err := fmt.Fprintln(os.Stderr, err.Error())
		if err != nil {
			fmt.Println("could not write to stderr")
		}
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(fmt.Errorf("error loading last version: %w", err))
		}
	}

	return app
}

func (app *SimApp) setAnteHandler(txConfig client.TxConfig) {
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			ante.HandlerOptions{
				Environment:              runtime.NewEnvironment(nil, app.logger, runtime.EnvWithMsgRouterService(app.MsgServiceRouter()), runtime.EnvWithQueryRouterService(app.GRPCQueryRouter())), // nil is set as the kvstoreservice to avoid module access
				AccountAbstractionKeeper: app.AccountsKeeper,
				AccountKeeper:            app.AuthKeeper,
				BankKeeper:               app.BankKeeper,
				ConsensusKeeper:          app.ConsensusParamsKeeper,
				FeegrantKeeper:           app.FeeGrantKeeper,
				SignModeHandler:          txConfig.SignModeHandler(),
				SigGasConsumer:           ante.DefaultSigVerificationGasConsumer,
			},
			app.IBCKeeper,
		},
	)
	if err != nil {
		panic(err)
	}

	// Set the AnteHandler for the app
	app.SetAnteHandler(anteHandler)
}

func (app *SimApp) setPostHandler() {
	postHandler, err := posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
	if err != nil {
		panic(err)
	}

	app.SetPostHandler(postHandler)
}

// Name returns the name of the App
func (app *SimApp) Name() string { return app.BaseApp.Name() }

// PreBlocker application updates every pre block
func (app *SimApp) PreBlocker(ctx sdk.Context, _ *abci.FinalizeBlockRequest) error {
	return app.ModuleManager.PreBlock(ctx)
}

// BeginBlocker application updates every begin block
func (app *SimApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *SimApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

// Configurator returns the configurator for the app
func (app *SimApp) Configurator() module.Configurator {
	return app.configurator
}

// InitChainer application update at chain initialization
func (app *SimApp) InitChainer(ctx sdk.Context, req *abci.InitChainRequest) (*abci.InitChainResponse, error) {
	var genesisState GenesisState
	err := json.Unmarshal(req.AppStateBytes, &genesisState)
	if err != nil {
		return nil, err
	}
	err = app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	if err != nil {
		return nil, err
	}
	return app.ModuleManager.InitGenesis(ctx, genesisState)
}

// LoadHeight loads a particular height
func (app *SimApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *SimApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns SimApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *SimApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns SimApp's InterfaceRegistry
func (app *SimApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns SimApp's TxConfig
func (app *SimApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *SimApp) DefaultGenesis() map[string]json.RawMessage {
	return app.ModuleManager.DefaultGenesis()
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetStoreKeys returns all the stored store keys.
func (app *SimApp) GetStoreKeys() []storetypes.StoreKey {
	keys := make([]storetypes.StoreKey, 0, len(app.keys))
	for _, key := range app.keys {
		keys = append(keys, key)
	}

	return keys
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *SimApp) SimulationManager() *module.SimulationManager {
	return app.simulationManager
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *SimApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register new CometBFT queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	app.ModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *SimApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *SimApp) RegisterTendermintService(clientCtx client.Context) {
	cmtApp := server.NewCometABCIWrapper(app)
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		cmtApp.Query,
	)
}

func (app *SimApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// ValidatorKeyProvider returns a function that generates a validator key
// Supported key types are those supported by Comet: ed25519, secp256k1, bls12-381
func (app *SimApp) ValidatorKeyProvider() runtime.KeyGenF {
	return func() (cmtcrypto.PrivKey, error) {
		return cmted25519.GenPrivKey(), nil
	}
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses(ac coreaddress.Codec) (map[string]bool, error) {
	modAccAddrs := make(map[string]bool)
	for acc := range GetMaccPerms() {
		addr, err := ac.BytesToString(authtypes.NewModuleAddress(acc))
		if err != nil {
			return nil, err
		}
		modAccAddrs[addr] = true
	}

	// allow the following addresses to receive funds
	govAddr, err := ac.BytesToString(authtypes.NewModuleAddress(govtypes.ModuleName))
	if err != nil {
		return nil, err
	}

	ibcMockAddr, err := ac.BytesToString(authtypes.NewModuleAddress(ibcmock.ModuleName))
	if err != nil {
		return nil, err
	}

	delete(modAccAddrs, govAddr)
	delete(modAccAddrs, ibcMockAddr)

	return modAccAddrs, nil
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// TODO: ibc module subspaces can be removed after migration of params
	// https://github.com/cosmos/ibc-go/issues/2010
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)

	return paramsKeeper
}

// IBC TestingApp functions

// GetBaseApp implements the TestingApp interface.
func (app *SimApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetStakingKeeper implements the TestingApp interface.
func (app *SimApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *SimApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetTxConfig implements the TestingApp interface.
func (app *SimApp) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *SimApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}
