# TransferAuthorization

`TransferAuthorization` implements the `Authorization` interface for `ibc.applications.transfer.v1.Msg`. It allows a granter to grant a grantee the privilege to submit MsgTransfer on its behalf. Please see the [Cosmos SDK docs](https://docs.cosmos.network/v0.47/modules/authz) for more details on granting privileges via the `Authz` module.

More specifically, the granter allows the grantee to transfer funds that belong to the granter on a set of 1 or more source port ID/channel ID pairs.

For each source port ID/channel ID pair, the granter shall be able to specify a spend limit for each denomination they wish to allow the grantee to be able to transfer.

The granter shall be able to specify the list of addresses that they allow to receive funds. If empty, then all addresses are allowed.


It takes: 

- a range of `SourcePorts` and a range of `SourceChannels` which together comprise the unique transfer channel identifiers over which authorized funds can be transferred.

- a range of (positive) `SpendLimits` that specify the maximum amount of tokens the grantee can spend. The `SpendLimit` is updated as the tokens are spent. This `SpendLimit` may also be updated to increase or decrease the limit as the granter wishes.

- an `AllowedAddrs` list that specifies the list of addresses that are allowed to receive funds. If this list is empty, then all addresses are allowed to receive funds from the `TransferAuthorization`.

Below is the `TransferAuthorization` message:

```protobuf
message TransferAuthorization {

  option (cosmos_proto.implements_interface) = "Authorization";

  // port and channel amounts

  repeated PortChannelAmount allocations = 1 [(gogoproto.nullable) = false];

}

message PortChannelAmount {

  // the port on which the packet will be sent
  string source_port = 1 [(gogoproto.moretags) = "yaml:\"source_port\""];

  // the channel by which the packet will be sent
  string source_channel = 2 [(gogoproto.moretags) = "yaml:\"source_channel\""];

  // spend limitation on the channel
  repeated cosmos.base.v1beta1.Coin spend_limit = 3
      [(gogoproto.nullable) = false, (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins"];

  // allowed addresses to be sent via transfer message
  repeated string allowed_addresses = 4;

}
```