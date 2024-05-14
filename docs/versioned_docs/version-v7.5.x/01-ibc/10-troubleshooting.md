---
title: Troubleshooting
sidebar_label: Troubleshooting
sidebar_position: 10
slug: /ibc/troubleshooting
---

# Troubleshooting

## Unauthorized client states

If it is being reported that a client state is unauthorized, this is due to the client type not being present
in the [`AllowedClients`](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/core/02-client/types/client.pb.go#L345) array.

Unless the client type is present in this array, all usage of clients of this type will be prevented.
