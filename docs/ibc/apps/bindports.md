<!--
order: 3
-->

# Bind ports

Learn what changes to make to bind modules to their ports on initialization. {synopsis}

Currently, ports must be bound on app initialization. In order to bind modules to their respective ports on initialization, the following needs to be implemented:

1. In the `module.go` file, add:

   ```go
   var (
       _ module.AppModule      = AppModule{}
       _ module.AppModuleBasic = AppModuleBasic{}
       // Add this line
       _ porttypes.IBCModule   = IBCModule{}
   )
   ```

1. Add port ID to the `GenesisState` proto definition:

   ```protobuf
   message GenesisState {
        string port_id = 1;
        // other fields
   }
   ```

1. Add port ID to `x/<moduleName>/types/genesis.go`:

   ```go
   // in x/<moduleName>/types/genesis.go

   // DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
   func DefaultGenesisState() *GenesisState {
       return &GenesisState{
           PortId:      PortID,
           // additional k-v fields
       }
   }

   // Validate performs basic genesis state validation returning an error upon any
   // failure.
   func (gs GenesisState) Validate() error {
       if err := host.PortIdentifierValidator(gs.PortId); err != nil {
           return err
       }
       //addtional validations

       return gs.Params.Validate()
   }
   ```

1. Bind to port(s) in `InitGenesis`:

   ```go
   // InitGenesis initializes the ibc-module state and binds to PortID.
   func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
       k.SetPort(ctx, state.PortId)

       // ...

       // Only try to bind to port if it is not already bound, since we may already own
       // port capability from capability InitGenesis
       if !k.IsBound(ctx, state.PortId) {
           // transfer module binds to the transfer port on InitChain
           // and claims the returned capability
           err := k.BindPort(ctx, state.PortId)
           if err != nil {
               panic(fmt.Sprintf("could not claim port capability: %v", err))
           }
       }

       // ...
   }
   ```

   With:

   ```go
   // IsBound checks if the  module is already bound to the desired port
   func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
       _, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
       return ok
   }

   // BindPort defines a wrapper function for the port Keeper's function in
   // order to expose it to module's InitGenesis function
   func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
       cap := k.portKeeper.BindPort(ctx, portID)
       return k.ClaimCapability(ctx, cap, host.PortPath(portID))
   }
   ```

   The module binds to the desired port(s) and returns the capabilities.
