/*
Package celestia implements a concrete LightClientModule and ClientState for
the Celestia light client. The Celestia light client tracks the latest header
consensus state from a Tendermint chain, but the consensus state that it
stores uses the data hash as the commitment root, instead of the app hash, as
done in vanilla Tendermint light clients. This light client is meant to be used
as a light client to verify proof of inclusion of header and block data of Rollkit
rollups in the Celestia data availability layer.
*/
package celestia
