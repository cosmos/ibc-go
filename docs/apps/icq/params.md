<!--
order: 7
-->

# Parameters

The IBC transfer application module contains the following parameters:

| Key              | Type     | Default Value |
|------------------|----------|---------------|
| `HostEnabled`    | bool     | `true`        |
| `AllowQueries`   | []string | `[]`          |

## `HostEnabled`

The `HostEnabled` parameter controls a chains ability to service ICQ host specific logic. This includes the following ICS-26 callback handlers:
- `OnChanOpenTry`
- `OnChanOpenConfirm`
- `OnChanCloseConfirm`
- `OnRecvPacket`

## `AllowQueries`

The `AllowQueries` parameter provides the ability for a chain to limit the queries that are authorized to be performed by defining an allowlist using the ABCI query path format.

For example, a Cosmos SDK based chain that wants to give permission to other chains to query balances of an account will define its parameters as follows:

```
"params": {
    "host_enabled": true,
    "allow_queries": ["/cosmos.bank.v1beta1.Query/AllBalances"]
}
```