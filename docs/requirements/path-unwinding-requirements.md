<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

The implementation of fungible token path unwinding vastly simplifies token transfers for end users. End users are unlikely to understand ibc denominations in great detail, and the consequences a direct transfer from chain A, to chain B can have on the fungibility of a token at the destination chain B, when the token sent is not a native or originating token from chain A, and is native to another chain, e.g. chain C.

Path Unwinding reduces the complexity of token transfer for the end user; a user simply needs to choose the final destination for their tokens and the complexity of determining the optimal route is abstracted away. This is a huge user experience improvement. 


## Problem

<!-- This section describes the problem that needs to be solved or the process that needs to be improved, as well as the environment in which the system will be used. This section could include a comparative evaluation of existing products, indicating why the proposed product is attractive and the advantages it provides. Describe the problems that cannot currently be solved without the envisioned solution. Show how it aligns with ecosystem trends, technology evolution, or strategic directions. List any other technologies, processes, or resources required to provide a complete solution. -->

A fungible token A transferred from chain A to chain B is an ibc denomination at chain B, where the ibc denom trace records the path the token has travelled to reach its destination chain. 

<!-- add in more detail about the denom trace more precisely -->

A user now wants to send this ibc denomination of token A, originating from chain A, onto another chain, chain C. If a user transfers token A on chain B directly to chain C, it will not be fungible with token A sent directly from chain A to chain B. This is because the ibc denomination of token A on chain C is different in both cases due to token A travelling along different paths to reach the same destination. This is the most simple case of the problem involving only 3 chains.

However, this problem is prevalent within the ecosystem and there are cases of ibc denominations on chains with >2 hops in the path. 

<!-- add in more detail about the number of > 2 hop denoms for a specific token, e.g. ATOM -->

## Objectives

To enable end users to automatically and atomically unwind fungible tokens when they specify a destination chain, so that tokens arrive at the destination chain with only 1 hop in the path. 

<!-- Summarize the important benefits the product will provide in a quantitative and measurable way. Platitudes (become recognized as a world-class <whatever>) and vaguely stated improvements (provide a more rewarding customer experience) are neither helpful nor verifiable. -->

## Scope

<!-- List the product's major features or capabilities. Think about how users will use the features, to ensure that the list is complete and that it does not include unnecessary features that sound interesting but don't provide value. Optionally, give each feature a unique and persistent label to permit tracing it to other system elements. List any product capabilities or characteristics that a stakeholder might expect but that are not planned for inclusion in the product or in a specific release. List items that were cut from scope, so the scope decision is not forgotten. -->

| Features  | Release |
| --------- | ------- |
| Automatic and atomic path unwinding for fungible tokens suitable for end users initiating a transfer | v7.3.0 |

# User requirements

## Use cases

### 1. Moving non-native assets between DeFi opportunities on different chains

Users transfer tokens from an origin chain to a DeFi chain to benefit from yield opportunities or other use cases on that chain, different from the origin chain. A better yield opportunity could then arise and a user would want to move the tokens to another chain to take advantage of this opportunity. Rather than having to manually route the tokens back through the originating chain onto the new chain, it would be much simpler if they could only be concerned with the final destination they want the tokens to arrive at. 

For example, ATOM is native to the Cosmos Hub, a user could transfer ATOM to Osmosis and deposit in a liquidity pool to earn yield on this token. After depositing their ATOM into a pool on Osmosis, a better yield opportunity could arise, for example a better pool APY on another Cosmos DEX, e.g. Crescent or a better yield on lending ATOM on Umee. The user would then want to transfer the ATOM from Osmosis to Crescent (or Umee).

### 2. Transferring liquid staking derivatives

Liquid staking derivatives are minted on liquid staking zones and represent a staked asset. There is a common misconception from users that these derivatives originate on the chain of the original staked token. This results in users sending derivatives back to the chain of the natively staked token and then onto the next destination. 

For examples, a user has stATOM on Osmosis, they want to move the stATOM to Evmos, instead of going from Osmosis --> Stride --> Evmos, a user tries to unwind themself and routes the tokens Osmosis --> Cosmos Hub --> Evmos 

### 3. Moving a token that originated from an interoperability zone or chain, or asset issuer

Tokens that originate from other blockchain ecosystems that don't yet support IBC, flow into the Cosmos ecosystem through interoperability zones. These tokens are then sent onto other chains with a specific use case for these tokens and a user could want to move this token from one chain to another. 

For example, ETH from Ethereum flows into Osmosis via Axelar, where the final step moving ETH from Ethereum to Osmosis uses an ICS-20 transfer from Axelar to Osmosis. A user may then want to move ETH from Osmosis onto another chain, for example Injective. However, the path with 1 hop would be a transfer from Axelar to Injective


<!-- A use case describes a sequence of interactions between a system and an external actor that results in the actor being able to achieve some outcome of value. An actor is a person (or sometimes another software system or a hardware device) that interacts with the system to perform a use case. Identify the various user classes that will use the feature. -->

# Functional requirements

<!-- They should describe as completely as necessary the system's behaviors under various conditions. They describe what the engineers must implement to enable users to accomplish their tasks (user requirements), thereby satisfying the business requirements. Software engineers don't implement business requirements or user requirements. They implement functional requirements, specific bits of system behavior. Each requirement should be uniquely identified with a meaningful tag. -->

## Assumptions and dependencies

1. A functional relayer implementation is required for this feature
2. There must be an on-chain registry on source chains that want to use path unwinding which contains information about possible ibc paths from this chain. As a minimum, information for all channels with `portID` relevant for ICS-20 and the `chainID` and `channelID` of connections from the chain. This is to enable routing of the final hop, from source to final destination without relying on an off-chain configuration. 
3. The feature will have no impact on the existing function of a typical ICS-20 transfer where a native token is sent to another chain
4. Fees will be treated the same as with other IBC applications


<!-- List any assumed factors that could affect the requirements. The project could be affected if these assumptions are incorrect, are not shared, or change. Also identify any dependencies the project has on external factors. -->

## Features

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 

# External interface requirements

<!-- They describe the interfaces to other software systems, hardware components, and users. Ideally they should state the purpose, format and content of messages used for input and output. -->

# Non-functional requirements

<!-- Other-than-functional requirements that do not specify what the system does, but rather how well it does those things. For example: quality requirements: performance, security, portability, etc. -->