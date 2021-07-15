<!--
order: 1
-->

# Types

## IBCAccount

IBCAccount implements the standard AccountI interface similar to Cosmos SDK x/auth module's BaseAccount or Module Account

```proto
// IBCAccount defines an account to which other chains have privileges
message IBCAccount {
    option (gogoproto.goproto_getters)         = false;
    option (gogoproto.goproto_stringer)        = false;
    option (cosmos_proto.implements_interface) = "IBCAccountI";

    cosmos.auth.v1beta1.BaseAccount base_account    = 1 [(gogoproto.embed) = true, (gogoproto.moretags) = "yaml:\"base_account\""];
    string sourcePort = 2;
    string sourceChannel = 3;
    string destinationPort = 4;
    string destinationChannel = 5;
}
```

As shown above, IBCAccount embeds the BaseAccount, and is assigned an Address and AccountNumber similar to the BaseAccount. However, because IBCAccount was designed to be used through the module, and not the user, there is no need to designate a PubKey or Sequence (which is implemented in the ModuleAccount).

Also, IBCAccount stores the information on the IBC Port and Channel that requested the creation of the account.

One can check if a specific address is an IBCAccount as well as the actual IBCAccount type through the IBCAccountKeeper's GetIBCAccount method. If the address queried doesn't exist or is not an IBC account, an error is returned.
