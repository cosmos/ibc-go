<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

<!-- They describe why the organization is implementing the product or feature (the benefits the organization hopes to achieve). They provide a reference for making decisions about proposed requirement changes and enhancements (i.e. decide if a proposed requirement is in or out of scope). Business requirements directly influence which user or functional requirements to implement and in what sequence. -->

<!-- Provide a high-level, short description of the software being specified and its purpose, including relevant benefits, objectives, and goals. Relate the software to ecosystem goals or strategies. -->

Using IBC as a mean of commuincating between chains and ecosystems has proven to be useful within Cosmos. There is then value in extending
this feature into other ecosystems, bringing a battle tested protocol of trusted commuincation as an option to send assets and data.

This is especially useful to protocols and companies whose business model is to improve cross-chain user interface, or to enable it when
it's not. The main use case for this is bridging assets between chains. There are multiple protocols and companies currently performing such
a service but none has yet been able to do it using IBC outside of the Cosmos ecosystem.

A core piece for this to happen is to have a light client implementation of each ecosystem that has to be integrated, and uses a **new** consensus
algorithm. This module broadens the horizon of light client development to not be limited to using Golang only for chains wanting use IBC and `ibc-go`,
but instead expands the choice to any programming language and toolchain that is able to compile to wasm instead.

Bridging assets, is likely the simplest for of interchain communication. Its value is confirmed on a daily basis, when considering the volumes that protocols
like [Axelar](https://dappradar.com/multichain/defi/axelar-network), Gravity, [Wormhole](https://dappradar.com/multichain/defi/wormhole/) and
[Layer0](TODO: add source for volume?) process. TODO: add sources for volume


## Problem

<!-- This section describes the problem that needs to be solved or the process that needs to be improved, as well as the environment in which the system will be used. This section could include a comparative evaluation of existing products, indicating why the proposed product is attractive and the advantages it provides. Describe the problems that cannot currently be solved without the envisioned solution. Show how it aligns with ecosystem trends, technology evolution, or strategic directions. List any other technologies, processes, or resources required to provide a complete solution. -->

In order to export IBC outside of Tendermint based ecosystems, there is a need to introduce new light clients. This is a core need for
companies and protocols trying to bridge ecosystems such as Ethereum, NEAR, Polkadot, etc. as none of these uses Tendermint as their
consensus mechanism. Introducing a new light client implementation is not straightforwrd. The implementor needs to follow the light client's
specification, and will try to make use of all available tools to keep the development cost reasonable.

Normally, most of available tools to implement a light client stem from the blockchain ecosystem this client belongs to. Say for example, if a developer
wants to implement the Polkadot finality gadget called GRANDPA, she will find that most of the tools are available on Substrate. Hence, being able to have a way
to let developers implement these light clients using the best and most accessible tools for the job is very beneficial, as it aavoids having to re implement
features that are otherwise available and likely heavily audited already. And since WASM is a well supported target that most programming languages support,
it becomes a proper solution to port the code for the `ibc-go` to interpret without requiring the entire light client being written using Go. 


## Objectives

<!-- Summarize the important benefits the product will provide in a quantitative and measurable way. Platitudes (become recognized as a world-class <whatever>) and vaguely stated improvements (provide a more rewarding customer experience) are neither helpful nor verifiable. -->

The objective of this module is to have allow two chains with heterogenous consensus algorithms being connected through light clients that are not necesarily written in Go, but
compiled to WASM instead.


## Scope

<!-- List the product's major features or capabilities. Think about how users will use the features, to ensure that the list is complete and that it does not include unnecessary features that sound interesting but don't provide value. Optionally, give each feature a unique and persistent label to permit tracing it to other system elements. List any product capabilities or characteristics that a stakeholder might expect but that are not planned for inclusion in the product or in a specific release. List items that were cut from scope, so the scope decision is not forgotten. -->

| Features              |  Release |
|---------------------- |----------|
| Dispatch messages to a|  v1      |
| light client written  |          |
| in wasm following the |          |
| `ClientState`         |          |
| interface             |          |


# User requirements

## Use cases

<!-- A use case describes a sequence of interactions between a system and an external actor that results in the actor being able to achieve some outcome of value. An actor is a person (or sometimes another software system or a hardware device) that interacts with the system to perform a use case. Identify the various user classes that will use the feature. -->

The first use case that this module will enable is the connection between GRANDPA light client chains and Tendermint light client chains. Further implementation of other light clients, such as NEAR, Ethereum, etc.
will likely consider building on top of this module.

# Functional requirements

<!-- They should describe as completely as necessary the system's behaviors under various conditions. They describe what the engineers must implement to enable users to accomplish their tasks (user requirements), thereby satisfying the business requirements. Software engineers don't implement business requirements or user requirements. They implement functional requirements, specific bits of system behavior. Each requirement should be uniquely identified with a meaningful tag. -->

The scope of this feature is to allow any implemention written in WASM to be compliant with the interface expressed
in [02-client ClientState interface](../../modules/core/exported/client.go).

## Assumptions and dependencies

<!-- List any assumed factors that could affect the requirements. The project could be affected if these assumptions are incorrect, are not shared, or change. Also identify any dependencies the project has on external factors. -->

This feature expects the [02-client refactor completed](https://github.com/cosmos/ibc-go/milestone/16), which is enabled in `ibc-go v7`.

## Features

<!-- Use a table like the following for the requirements:
| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
-->

# External interface requirements

<!-- They describe the interfaces to other software systems, hardware components, and users. Ideally they should state the purpose, format and content of messages used for input and output. -->

# Non-functional requirements

<!-- Other-than-functional requirements that do not specify what the system does, but rather how well it does those things. For example: quality requirements: performance, security, portability, etc. -->