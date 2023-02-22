<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

Rather than create a new channel to expand upon the capabilities of an existing channel, channel upgradability enables new features and capabilities to be added to existing channels. 

## Problem

IBC is designed so that a specific application module will claim the capability of a channel. Currently, once a channel is opened and the channel handshake is complete, you cannot change the application module claiming that channel. This means that if you wanted to upgrade an existing application module or add middleware to both ends of a channel, you would need to open a new channel with these new modules meaning all previous state in the prior channel would be lost. This is particularly important for channels using the ics-20 (fungible token transfer) application module because tokens are not fungible between channels.

## Objectives

To enable existing channels to upgrade the application module claiming the channel or add middleware to both ends of an existing channel whilst retaining the state of the channel. 

## Scope

<!-- List the product's major features or capabilities. Think about how users will use the features, to ensure that the list is complete and that it does not include unnecessary features that sound interesting but don't provide value. Optionally, give each feature a unique and persistent label to permit tracing it to other system elements. List any product capabilities or characteristics that a stakeholder might expect but that are not planned for inclusion in the product or in a specific release. List items that were cut from scope, so the scope decision is not forgotten. -->

| Features  | Release |
| --------- | ------- |
| Performing a channel upgrade results in an application module changing from v1 to v2, claiming the same `channelID` and `portID` | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` changing the ordering from a higher to lower degree of ordering | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` having additional middleware added to the application stack on both sides of the channel | v1 |


# User requirements

## Use cases

Upgrading an existing application module from v1 to v2, e.g. new features could be added to the existing ics-20 application module which would result in a new version of the module.

Adding middleware on both sides of an existing channel, e.g. relayer incentivisation middleware, ics-29, requires middleware to be added to both ends of a channel to incentivise the `recvPacket`, `acknowledgePacket` and `timeoutPacket`.

# Functional requirements

<!-- They should describe as completely as necessary the system's behaviors under various conditions. They describe what the engineers must implement to enable users to accomplish their tasks (user requirements), thereby satisfying the business requirements. Software engineers don't implement business requirements or user requirements. They implement functional requirements, specific bits of system behavior. Each requirement should be uniquely identified with a meaningful tag. -->

## Assumptions and dependencies

<!-- List any assumed factors that could affect the requirements. The project could be affected if these assumptions are incorrect, are not shared, or change. Also identify any dependencies the project has on external factors. -->

Functional relayer infrastructure is required to perform a channel upgrade.

## Features

<!-- Use a table like the following for the requirements:
| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
-->

# External interface requirements

<!-- They describe the interfaces to other software systems, hardware components, and users. Ideally they should state the purpose, format and content of messages used for input and output. -->

# Non-functional requirements

<!-- Other-than-functional requirements that do not specify what the system does, but rather how well it does those things. For example: quality requirements: performance, security, portability, etc. -->