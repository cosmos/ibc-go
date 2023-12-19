---
slug: /params.md
---

# Parameters

## 02-Client

The 02-client submodule contains the following parameters:

| Key              | Type     | Default Value                                     |
| ---------------- | -------- | ------------------------------------------------- |
| `AllowedClients` | []string | `"06-solomachine","07-tendermint","09-localhost"` |

### AllowedClients

The allowed clients parameter defines an allow list of client types supported by the chain. A client
that is not registered on this list will fail upon creation or on genesis validation. Note that,
since the client type is an arbitrary string, chains must not register two light clients which
return the same value for the `ClientType()` function, otherwise the allow list check can be
bypassed. If `AllowAllClients` wildcard (`*`) is set, then all client type are supported.
