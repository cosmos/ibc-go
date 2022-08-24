<!--
order: 3
-->

# Integration

Learn how to integrate ICQ host functionality to your chain and send query requests from your module. The following document only applies for Cosmos SDK chains. {synopsis}

The ICQ module itself is responsible for host functionalities such as receiving the packets from other chains, validate and perform the queries and send the response as acknowledgement.

Modules on other chains which wish to query the host chain should implement IBCModule interface and are responsible for sending `InterchainQueryPacket`. 


### Example integration

```go
// app.go

// Register the AppModule for the ICQ module and the
ModuleBasics = module.NewBasicManager(
    ...
    icq.AppModuleBasic{},
    ...
)

... 

// Add module account permissions for the ICQ module
maccPerms = map[string][]string{
    ...
    icqtypes.ModuleName:            nil,
}

...

// Add ICQ Keeper
type App struct {
    ...

    ICQKeeper       icqkeeper.Keeper

    ...
}

...

// Create store keys for Keeper
keys := sdk.NewKVStoreKeys(
    ...
    icqtypes.StoreKey,
    ...
)

... 

// Create the scoped keeper
scopedICQKeeper := app.CapabilityKeeper.ScopeToModule(icqtypes.ModuleName)

...

app.ICQKeeper = icqkeeper.NewKeeper(
		appCodec, keys[icqtypes.StoreKey], app.GetSubspace(icqtypes.ModuleName),
		app.IBCKeeper.ChannelKeeper, app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		scopedICQKeeper, app.BaseApp,
)

// Create ICQ AppModule
icqModule := icq.NewAppModule(&app.ICQKeeper)

// Create IBC Module
icqIBCModule := icq.NewIBCModule(app.ICQKeeper)

// Register host and authentication routes
ibcRouter.AddRoute(icqtypes.ModuleName, icqIBCModule)

...

// Register ICQ AppModule's
app.moduleManager = module.NewManager(
    ...
    icqModule,
)

...

// Add ICQ module InitGenesis logic
app.mm.SetOrderInitGenesis(
    ...
    icqtypes.ModuleName,
    ...
)
```

### Sending Packets

In order to send a packet from a module, first you need to prepare your query `Data` and encapsule it in an ABCI `RequestQuery`. Then you can use `SerializeCosmosQuery` to construct the `Data` of `InterchainQueryPacketData` packet data. After these steps you should use ICS 4 wrapper to send your packet to the host chain through a valid channel.

```go
q := banktypes.QueryAllBalancesRequest{
    Address: "cosmos1tshnze3yrtv3hk9x536p7znpxeckd4v9ha0trg",
    Pagination: &query.PageRequest{
        Offset: 0,
        Limit: 10,
    },
}
reqs := []abcitypes.RequestQuery{
	{
		Path: "/cosmos.bank.v1beta1.Query/AllBalances",
		Data: k.cdc.MustMarshal(&q),
	},
}

bz, err := icqtypes.SerializeCosmosQuery(reqs)
if err != nil {
	return 0, err
}
icqPacketData := icqtypes.InterchainQueryPacketData{
	Data: bz,
}

packet := channeltypes.NewPacket(
		icqPacketData.GetBytes(),
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		clienttypes.ZeroHeight(),
		timeoutTimestamp,
)

// Send the `packet` with ICS-4 interface
```

### Response Acknowledgment

Successful acknowledgment will be sent back to querier module as `InterchainQueryPacketAck`. The `Data` field should be deserialized to and array of ABCI `ResponseQuery` with `DeserializeCosmosResponse` function. Responses are sent in the same order as the requests.

```go
switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		var ackData icqtypes.InterchainQueryPacketAck
		if err := icqtypes.ModuleCdc.UnmarshalJSON(resp.Result, &ackData); err != nil {
			return sdkerrors.Wrap(err, "failed to unmarshal interchain query packet ack")
		}

        resps, err := icqtypes.DeserializeCosmosResponse(ackData.Data)
        if err != nil {
            return sdkerrors.Wrap(err, "failed to unmarshal interchain query packet ack to cosmos response")
        }

		if len(resps) < 1 {
			return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "no responses in interchain query packet ack")
		}

		var r banktypes.QueryAllBalancesResponse
		if err := k.cdc.Unmarshal(resps[0].Value, &r); err != nil {
			return sdkerrors.Wrapf(err, "failed to unmarshal interchain query response to type %T", resp)
		}

        // `r` is the response of your query
...
```