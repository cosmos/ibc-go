package simapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik" // this is used for serving docs
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/store/streaming"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata_pulsar"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"

	ica "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v6/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v6/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v6/modules/apps/29-fee/types"
	transfer "github.com/cosmos/ibc-go/v6/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v6/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v6/modules/core"
	ibcclientclient "github.com/cosmos/ibc-go/v6/modules/core/02-client/client"
	porttypes "github.com/cosmos/ibc-go/v6/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v6/modules/core/keeper"
	ibcmock "github.com/cosmos/ibc-go/v6/testing/mock"
	simappupgrades "github.com/cosmos/ibc-go/v6/testing/simapp/upgrades"
	v6 "github.com/cosmos/ibc-go/v6/testing/simapp/upgrades/v6"
	ibctestingtypes "github.com/cosmos/ibc-go/v6/testing/types"
)

// IBC application testing ports
const (
	MockFeePort string = ibcmock.ModuleName + ibcfeetypes.ModuleName
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
				upgradeclient.LegacyProposalHandler,
				upgradeclient.LegacyCancelProposalHandler,
				ibcclientclient.UpdateClientProposalHandler,
				ibcclientclient.UpgradeProposalHandler,
			},
		),
		groupmodule.AppModuleBasic{},
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ibcmock.AppModuleBasic{},
		ica.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		vesting.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
	)
)

var (
	_ runtime.AppI            = (*SimApp)(nil)
	_ servertypes.Application = (*SimApp)(nil)
)

// SimApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type SimApp struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys map[string]*storetypes.KVStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	GroupKeeper           groupkeeper.Keeper
	NFTKeeper             nftkeeper.Keeper
	CrisisKeeper          crisiskeeper.Keeper
	UpgradeKeeper         upgradekeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCFeeKeeper          ibcfeekeeper.Keeper
	ICAControllerKeeper   icacontrollerkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper
	ScopedFeeMockKeeper       capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedIBCMockKeeper       capabilitykeeper.ScopedKeeper
	ScopedICAMockKeeper       capabilitykeeper.ScopedKeeper

	// make IBC modules public for test purposes
	// these modules are never directly routed to by the IBC Router
	ICAAuthModule ibcmock.IBCModule
	FeeMockModule ibcmock.IBCModule

	// simulation manager
	sm *module.SimulationManager
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
	var (
		app        = &SimApp{}
		appBuilder *runtime.AppBuilder

		// merge the AppConfig and other configuration in one config
		appConfig = depinject.Configs(
			AppConfig,
			depinject.Supply(
				// supply the application options
				appOpts,

				// For providing a custom inflation function for x/mint add here your
				// custom function that implements the minttypes.InflationCalculationFn
				// interface.

				// For providing a custom authority to a module simply add it below. By
				// default the governance module is the default authority.
				//
				// map[string]sdk.AccAddress{
				// 	minttypes.ModuleName: authtypes.NewModuleAddress(authtypes.ModuleName),
				// },
			),
		)
	)

	if err := depinject.Inject(appConfig,
		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AccountKeeper,
		&app.BankKeeper,
		&app.CapabilityKeeper,
		&app.StakingKeeper,
		&app.SlashingKeeper,
		&app.MintKeeper,
		&app.DistrKeeper,
		&app.GovKeeper,
		&app.CrisisKeeper,
		&app.UpgradeKeeper,
		&app.ParamsKeeper,
		&app.AuthzKeeper,
		&app.EvidenceKeeper,
		&app.FeeGrantKeeper,
		&app.GroupKeeper,
		&app.IBCKeeper,
		&app.IBCFeeKeeper,
		&app.ICAControllerKeeper,
		&app.ICAHostKeeper,
		&app.ScopedIBCKeeper,
		&app.ScopedTransferKeeper,
		&app.ScopedFeeMockKeeper,
		&app.ScopedICAControllerKeeper,
		&app.ScopedICAHostKeeper,
		&app.ScopedIBCMockKeeper,
		&app.NFTKeeper,
		&app.ConsensusParamsKeeper,
	); err != nil {
		panic(err)
	}

	app.App = appBuilder.Build(logger, db, traceStore, baseAppOptions...)

	// configure state listening capabilities using AppOptions
	// we are doing nothing with the returned streamingServices and waitGroup in this case
	if _, _, err := streaming.LoadStreamingServices(app.App.BaseApp, appOpts, app.appCodec, app.keys); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Create IBC Router
	ibcRouter := porttypes.NewRouter()

	// Middleware Stacks

	// Mock Module Stack

	// Mock Module setup for testing IBC and also acts as the interchain accounts authentication module
	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	mockModule := ibcmock.NewAppModule(&app.IBCKeeper.PortKeeper)

	// The mock module is used for testing IBC
	mockIBCModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(ibcmock.ModuleName, app.ScopedIBCMockKeeper))
	ibcRouter.AddRoute(ibcmock.ModuleName, mockIBCModule)

	// Create Transfer Stack
	// SendPacket, since it is originating from the application to core IBC:
	// transferKeeper.SendPacket -> fee.SendPacket -> channel.SendPacket

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
	// channel.RecvPacket -> fee.OnRecvPacket -> transfer.OnRecvPacket

	// transfer stack contains (from top to bottom):
	// - IBC Fee Middleware
	// - Transfer

	// create IBC module from bottom to top of stack
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)

	// Add transfer stack to IBC Router
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)

	// Create Interchain Accounts Stack
	// SendPacket, since it is originating from the application to core IBC:
	// icaAuthModuleKeeper.SendTx -> icaController.SendPacket -> fee.SendPacket -> channel.SendPacket

	// initialize ICA module with mock module as the authentication module on the controller side
	var icaControllerStack porttypes.IBCModule
	icaControllerStack = ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp("", app.ScopedICAMockKeeper))
	app.ICAAuthModule = icaControllerStack.(ibcmock.IBCModule)
	icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, app.ICAControllerKeeper)
	icaControllerStack = ibcfee.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper)

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket

	var icaHostStack porttypes.IBCModule
	icaHostStack = icahost.NewIBCModule(app.ICAHostKeeper)
	icaHostStack = ibcfee.NewIBCMiddleware(icaHostStack, app.IBCFeeKeeper)

	// Add host, controller & ica auth modules to IBC router
	ibcRouter.
		// the ICA Controller middleware needs to be explicitly added to the IBC Router because the
		// ICA controller module owns the port capability for ICA. The ICA authentication module
		// owns the channel capability.
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(ibcmock.ModuleName+icacontrollertypes.SubModuleName, icaControllerStack) // ica with mock auth module stack route to ica (top level of middleware stack)

	// Create Mock IBC Fee module stack for testing
	// SendPacket, since it is originating from the application to core IBC:
	// mockModule.SendPacket -> fee.SendPacket -> channel.SendPacket

	// OnRecvPacket, message that originates from core IBC and goes down to app, the flow is the otherway
	// channel.RecvPacket -> fee.OnRecvPacket -> mockModule.OnRecvPacket

	// OnAcknowledgementPacket as this is where fee's are paid out
	// mockModule.OnAcknowledgementPacket -> fee.OnAcknowledgementPacket -> channel.OnAcknowledgementPacket

	// create fee wrapped mock module
	feeMockModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(MockFeePort, app.ScopedFeeMockKeeper))
	app.FeeMockModule = feeMockModule
	feeWithMockModule := ibcfee.NewIBCMiddleware(feeMockModule, app.IBCFeeKeeper)
	ibcRouter.AddRoute(MockFeePort, feeWithMockModule)

	// Seal the IBC Router
	app.IBCKeeper.SetRouter(ibcRouter)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], &app.StakingKeeper, app.SlashingKeeper,
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	/****  Module Options ****/

	// Sets the version setter for the upgrade module
	app.UpgradeKeeper.SetVersionSetter(app.BaseApp)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	genesisModuleOrder := []string{
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		crisistypes.ModuleName,
		ibchost.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		ibcmock.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		group.ModuleName,
	}

	app.App.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.App.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	// Uncomment if you want to set a custom migration order here.
	// app.ModuleManager.SetOrderMigrations(custom order)

	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)

	// RegisterUpgradeHandlers is used for registering any on-chain upgrades.
	// Make sure it's called after `app.ModuleManager` and `app.configurator` are set.
	app.RegisterUpgradeHandlers()

	// add test gRPC service for testing gRPC queries in isolation
	testdata_pulsar.RegisterQueryServer(app.GRPCQueryRouter(), testdata_pulsar.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(app.keys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)

	if err := app.Load(loadLatest); err != nil {
		panic(err)
	}

	return app
}

// Name returns the name of the App
func (app *SimApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *SimApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.App.ModuleManager.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *SimApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.App.ModuleManager.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *SimApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.App.ModuleManager.GetVersionMap())
	return app.App.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *SimApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *SimApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		// do not add the following modules to blocked addresses
		// this is only used for testing
		if acc == ibcmock.ModuleName {
			continue
		}

		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// GetModuleManager returns the app module manager
// NOTE: used for testing purposes
func (app *SimApp) GetModuleManager() *module.Manager {
	return app.App.ModuleManager
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

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *SimApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// TestingApp functions

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

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *SimApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// GetTxConfig implements the TestingApp interface.
func (app *SimApp) GetTxConfig() client.TxConfig {
	return MakeTestEncodingConfig().TxConfig
}

// SimulationManager implements the SimulationApp interface
func (app *SimApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *SimApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *SimApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *SimApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(ctx client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)

	return paramsKeeper
}

// setupUpgradeHandlers sets all necessary upgrade handlers for testing purposes
func (app *SimApp) setupUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		simappupgrades.DefaultUpgradeName,
		simappupgrades.CreateDefaultUpgradeHandler(app.mm, app.configurator),
	)

	// NOTE: The moduleName arg of v6.CreateUpgradeHandler refers to the auth module ScopedKeeper name to which the channel capability should be migrated from.
	// This should be the same string value provided upon instantiation of the ScopedKeeper with app.CapabilityKeeper.ScopeToModule()
	// TODO: update git tag in link below
	// See: https://github.com/cosmos/ibc-go/blob/v5.0.0-rc2/testing/simapp/app.go#L304
	app.UpgradeKeeper.SetUpgradeHandler(
		v6.UpgradeName,
		v6.CreateUpgradeHandler(
			app.mm,
			app.configurator,
			app.appCodec,
			app.keys[capabilitytypes.ModuleName],
			app.CapabilityKeeper,
			ibcmock.ModuleName+icacontrollertypes.SubModuleName,
		),
	)
}
