# `TransferAuthorization`

`TransferAuthorization` implements the `Authorization` interface for `ibc.applications.transfer.v1.MsgTransfer`. It allows a granter to grant a grantee the privilege to submit MsgTransfer on its behalf. Please see the [Cosmos SDK docs](https://docs.cosmos.network/v0.47/modules/authz) for more details on granting privileges via the `x/authz` module.

More specifically, the granter allows the grantee to transfer funds that belong to the granter over a specified channel.

For the specified channel, the granter must be able to specify a spend limit of a specific denomination they wish to allow the grantee to be able to transfer.

The granter may be able to specify the list of addresses that they allow to receive funds. If empty, then all addresses are allowed.


It takes: 

- a `SourcePort` and a `SourceChannel` which together comprise the unique transfer channel identifier over which authorized funds can be transferred.

- a `SpendLimit` that specifies the maximum amount of tokens the grantee can spend. The `SpendLimit` is updated as the tokens are spent. This `SpendLimit` may also be updated to increase or decrease the limit as the granter wishes.

- an `AllowList` list that specifies the list of addresses that are allowed to receive funds. If this list is empty, then all addresses are allowed to receive funds from the `TransferAuthorization`.

Setting a `TransferAuthorization` is expected to fail if:
- the spend limit is nil
- the denomination of the spend limit is an invalid coin type
- the source port ID is invalid
- the source channel ID is invalid
- there are duplicate entries in the `AllowList`

Below is the `TransferAuthorization` message:

```go
func NewTransferAuthorization(allocations ...Allocation) *TransferAuthorization {
	return &TransferAuthorization{
		Allocations: allocations,
	}
}

type Allocation struct {
	// the port on which the packet will be sent
	SourcePort string 
	// the channel by which the packet will be sent
	SourceChannel string 
	// spend limitation on the channel
	SpendLimit sdk.Coins  
	// allow list of receivers, an empty allow list permits any receiver address
	AllowList []string 
}

```
