<!--
order: 1
-->

# Overview

Learn about what the ICQ module is, and how to send queries to modules on other chains and receive the acknowledgment {synopsis}


## What is the ICQ (Interchain Query) module?

ICQ is a protocol which enables the Cosmos-SDK IBC enabled modules to query other modules data/endpoints on other chains and receive the response of their query.

## Concepts 

`Host Chain`: The chain with icq module enabled which will receive query request packets from other chains process them and then sends the query response as an acknowledgment.

`Querier Chain`: The chain who wants to query specific piece of information from another chain which have to sends the query request to the host chain and receive the acknowledgment. 
	
## Considerations

This implementation commits the response of ABCI queries to state by sending them as IBC packets. So any changes in the query endpoint that changes the response for a particular request should be followed up by a chain upgrade plan.
