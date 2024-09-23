<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

The implementation of fungible token path unwinding vastly simplifies token transfers for end users. End users are unlikely to understand IBC denominations in great detail, and the consequences a direct transfer from chain A, to chain B can have on the fungibility of a token at the destination chain B, when the token sent is not a native or originating token from chain A, and is native to another chain, e.g. chain C.

Path unwinding reduces the complexity of token transfer for the end user; a user simply needs to choose the final destination for their tokens and the complexity of determining the optimal route is abstracted away. This is a huge user experience improvement.

In addition to unwinding, when a user receives their token on a destination chain, they then want to use the token in some way. By enabling token forwarding, a user can receive a token, perform some action with that token, for example a swap, and then send the token onto another chain. We observe that the complexity of IBC is increasingly being abstracted away from end users and automating workflows such as transfer, swap and forward with a single signed transaction significantly enhances usability. 

## Problem

A fungible token A transferred from chain A to chain B is an IBC denomination at chain B, where the IBC denom trace records the path the token has travelled to reach its destination chain. 

A user now wants to send this IBC denomination of token A, originating from chain A, onto another chain, chain C. If a user transfers token A on chain B directly to chain C, it will not be fungible with token A sent directly from chain A to chain C. This is because the IBC denomination of token A on chain C is different in both cases due to token A travelling along different paths to reach the same destination. This is the most simple case of the problem involving only 3 chains.

However, this problem is prevalent within the ecosystem and there are cases of IBC denominations on chains with >2 hops in the path. 

Regarding forwarding, if a user wants to transfer tokens between chains, then perform an action with those tokens, without forwarding, a user would have to sign each transaction on every chain and wait for the tokens to arrive at the destination before performing the next action. This is time consuming and a provides a poor user experience, a user also cannot just specify the desired outcome of their workflow in a trivial way.   

## Objectives

To enable end users to automatically and atomically unwind fungible tokens when they specify a destination chain, so that tokens arrive at the destination chain with only 1 hop in the path and to be able to forward the token to another destination after it has been unwound.

## Scope

| Features  | Release |
| --------- | ------- |
| Automatic and atomic path unwinding for fungible tokens suitable for end users initiating a transfer | v9.0.0 |
| Token forwarding for fungible tokens for end users initiating a transfer | v9.0.0 |

# User requirements

## Use cases

### 1. Moving non-native assets between DeFi opportunities on different chains

Users transfer tokens from an origin chain to a DeFi chain to benefit from yield opportunities or other use cases on that chain, different from the origin chain. A better yield opportunity could then arise and a user would want to move the tokens to another chain to take advantage of this opportunity. Rather than having to manually route the tokens back through the originating chain onto the new chain, it would be much simpler if they could only be concerned with the final destination they want the tokens to arrive at. 

For example, ATOM is native to the Cosmos Hub, a user could transfer ATOM to Osmosis and deposit in a liquidity pool to earn yield on this token. After depositing their ATOM into a pool on Osmosis, a better yield opportunity could arise, for example a better pool APY on another Cosmos DEX, e.g. Crescent or a better yield on lending ATOM on Umee. The user would then want to transfer the ATOM from Osmosis to Crescent (or Umee).

### 2. Transferring liquid staking derivatives

Liquid staking derivatives are minted on liquid staking zones and represent a staked asset. There is a common misconception from users that these derivatives originate on the chain of the original staked token. This results in users sending derivatives back to the chain of the natively staked token and then onto the next destination. 

For examples, a user has stATOM on Osmosis, they want to move the stATOM to Evmos, instead of going from Osmosis --> Stride --> Evmos, a user tries to unwind themself and routes the tokens Osmosis --> Cosmos Hub --> Evmos. 

### 3. Moving a token that originated from an interoperability zone or chain, or asset issuer

Tokens that originate from other blockchain ecosystems that don't yet support IBC, flow into the Cosmos ecosystem through interoperability zones. These tokens are then sent onto other chains with a specific use case for these tokens and a user could want to move this token from one chain to another. 

For example, ETH from Ethereum flows into Osmosis via Axelar, where the final step moving ETH from Ethereum to Osmosis uses an ICS-20 transfer from Axelar to Osmosis. A user may then want to move ETH from Osmosis onto another chain, for example Injective. However, the path with 1 hop would be a transfer from Axelar to Injective.

### 4. Moving a token from one chain to another, swapping the token and transferring it onwards to a new destination

A user on one chain, for example the Cosmos Hub holds an asset, e.g. ATOM and wants to instead have AKT on Akash. The user must transfer to ATOM to a DEX chain, swap the ATOM for AKT and then send the AKT onwards to Akash. 

# Functional requirements

## Assumptions and dependencies

1. A functional relayer implementation is required for this feature.
2. Routing information for the final hop for unwinding or for forwarding is configured for the user through a front end client, or configured by another means using off-chain data. 
3. The feature will have no impact on the existing function of a typical ICS-20 transfer where a native token is sent to another chain.
4. Fees are not within the scope of these requirements.
5. The functionality to enable a specific action before forwarding is not in scope of these requirements.
6. If a transfer contains multiple unique tokens, the unwinding and forwarding functionalities only need to support unwinding or forwarding through the same path, i.e. if there is a transfer containing a native token and 1 hop denom being sent from source to destination, both tokens would always go through the same path. In the future, it may be desirable for different tokens to be unwound through different paths, to support actions such as atomic transfer and supply liquidity to a liquidity pool, but it is currently out of scope for the first version of this feature. 

## Features

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 1.01 | When a user initiates a transfer to a destination chain with an IBC denom with > 1 hop, the token shall be sent back to its originating chain before being sent onto the destination as a user selected option | | `Draft` | 
| 1.02 | If a user wants to unwind tokens, then they can select this as option for the transfer | | `Draft` | 
| 1.03 | The unwinding shall completely succeed or the tokens are recoverable on the chain they were sent from by the user | | `Draft` | 
| 1.04 | When unwinding is used in combination with forwarding, both the unwind and forwarding should succeed or the tokens should be recoverable on the sending chain | | `Draft` | 
| 1.05 | The forwarding mechanism shall allow a user to transfer tokens beyond the first destination for those tokens | | `Draft` | 
| 1.06 | The forwarding mechanism shall allow tokens to have some action performed on them before being sent onto a new destination | | `Draft` | 
| 1.07 | The routing information for forwarding or to go from unwound token to destination must be input with the initial transfer | | `Draft` |
| 1.08 | If an intermediate chain does not have the unwinding or forwarding functionality, the tokens must be recoverable on the sending chain | | `Draft` | 
| 1.09 | If unwinding or forwarding fails, then the reason for the failure should be returned in an error | | `Draft` |
| 1.10 | When unwinding, it should be possible for the forwarding path to be evaluated implicitly from introspecting the denomination trace or to be explicitly input as forwarding hops by the user | | `Draft` |  
| 1.11 | When using unwinding in combination with x/authz, a granter can specify the allowed forwarding paths | | `Draft` |   

# External interface requirements

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 2.01 | There must be a CLI interface to initiate a transfer using path unwinding | | `Draft` | 
| 2.02 | There must be a CLI interface to initiate a transfer using forwarding | | `Draft` | 
| 2.03 | There must be a CLI interface to initiate a transfer using unwinding and forwarding in combination | | `Draft` | 

# Non-functional requirements

## Security

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 3.01 | It must not be possible for a users tokens to be intercepted by another actor during path-unwinding or token forwarding | | `Draft` |
