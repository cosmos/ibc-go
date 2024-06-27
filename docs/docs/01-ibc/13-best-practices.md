---
title: Best Practices
sidebar_label: Best Practices
sidebar_position: 12
slug: /ibc/best-practices
---

# Best practices

## Identifying legitimate channels

Identifying which channel to use can be difficult as it requires verifying information about the chains you want to connect to. 
Channels are based on a light client. A chain can be uniquely identifier by its chain-id, validator set pairing. It is unsafe to rely only on the chain-id. 
Any user can create a client with any chain-id, but only the chain with correct validator set and chain-id can produce headers which would update that client. 

Which channel to use is based on social consensus. The desired channel should have the following properities:
- based on a valid client (can only be updated by the chain it connects to)
- has sizable activity
- the underlying client is active

To verify if a client is valid. You will need to obtain a header from the chain you want to connect to. This can be done by running a full node for that chain or relying on a trusted grpc address. 
Then you should query the light client you want to verify and obtain its latest consensus state. All consensus state fields must match the header queried for at same height as the consensus state (root, timestamp, next validator set hash).  

Explorers and wallets are highly encouraged to follow this practice. It is unsafe to algorithmically add new channels without following this process. 
