---
title: Permissioning
sidebar_label: Permissioning
sidebar_position: 14
slug: /ibc/permissioning
---

# Permissioning

IBC is designed at its base level to be a permissionless protocol. This does not mean that chains cannot add in permissioning on top of IBC. In ibc-go this can be accomplished by implementing and wiring an ante-decorator that checks if the IBC message is signed by a permissioned authority. If the signer address check passes, the tx can go through; otherwise it is rejected from the mempool.

The antehandler runs before message-processing so it acts as a customizable filter that can reject messages before they get included in the block. The Cosmos SDK allows developers to write ante-decorators that can be stacked with others to add multiple independent customizable filters that run in sequence. Thus, chain developers that want to permission IBC messages are advised to implement their own custom permissioned IBC ante-decorator to add to the standard ante-decorator stack.

## Best practices

`MsgCreateClient`: permissioning the client creation is the most important for permissioned IBC. This will prevent malicious relayers from creating clients to fake chains. If a chain wants to control which chains are connected to it directly over IBC, the best way to do this is by controlling which clients get created. The permissioned authority can create clients only of counterparties that the chain approves of. The permissioned authority can be the governance account, however `MsgCreateClient` contains a consensus state that can be expired by the time governance passes the proposal to execute the message. Thus, if the voting period is longer than the unbonding period of the counterparty, it is advised to use a permissioned authority that can immediately execute the transaction (e.g. a trusted multisig).

`MsgConnectionOpenInit`: permissioning this message will give the chain control over the connections that are opened and also will control which connection identifier is associated with which counterparty.

`MsgConnectionOpenTry`: permissioning this message through a permissioned address check is ill-advised because it will prevent relayers from easily completing the handshake that was initialized on the counterparty. However, if the chain does want strict control of exactly which connections are opened, it can permission this message. Be aware, if two chains with strict permissions try to open a connection it may take much longer than expected.

`MsgChannelOpenInit`: permissioning this message will give the chain control over the channels that are opened and also will control which channel identifier is associated with which counterparty.

`MsgChannelOpenTry`: permissioning this message through a permissioned address check is ill-advised because it will prevent relayers from easily completing the handshake that was initialized on the counterparty. However, if the chain does want strict control of exactly which channels are opened, it can permission this message. Be aware, if two chains with strict permissions try to open a channel it may take much longer than expected.

It is not advised to permission any other message from ibc-go. Permissionless relayers should still be allowed to complete handshakes that were authorized by permissioned parties, and to relay user packets on channels that were also authorized by permissioned parties. This provides the maximum liveness provided by a permissionless relayer network with the safety guarantees provided by permissioned client, connection, and channel creation.

## Genesis setup

Chains that are starting up from genesis have the option of initializing authorized clients, connections and channels from genesis. This allows chains to automatically connect to desired chains with a desired identifier.

Note: The chain must be launched soon after the genesis file is created so that the client creation does not occur with an expired consensus state. The connections and channels must also simply have their `INIT` messages executed so that relayers can complete the rest of the handshake.
