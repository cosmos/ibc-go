---
slug: /params.md
---

# Parameters

## 02-Client

The 02-client submodule contains the following parameters:

| Key              | Type     | Default Value |
| ---------------- | -------- | ------------- |
| `AllowedClients` | []string | `"*"`         |

### AllowedClients

The allowed clients parameter defines an allow list of client types supported by the chain. The 
default value is a single-element list containing the `AllowAllClients` wildcard (`"*"`). When the
wildcard is used, then all client types are supported by default. Alternatively, the parameter
may be set with a list of client types (e.g. `"06-solomachine","07-tendermint","09-localhost"`).
A client type that is not registered on this list will fail upon creation or on genesis validation.
Note that, since the client type is an arbitrary string, chains must not register two light clients
which return the same value for the `ClientType()` function, otherwise the allow list check can be
bypassed.
