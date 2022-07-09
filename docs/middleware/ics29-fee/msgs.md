<!--
order: 3
-->

# Escrowing fees

The fee middleware module exposes two different ways to pay fees for relaying IBC packets:

1. `MsgPayPacketFee`, which enables the escrowing of fees for a packet at the next sequence send and should be combined into one `MultiMsgTx` with the message that will be paid for.  

    Note that the `Relayers` field has been set up to allow for an optional whitelist of relayers permitted to receive this fee, however, this feature has not yet been enabled at this time.

    ```
    type MsgPayPacketFee struct{
      // fee encapsulates the recv, ack and timeout fees associated with an IBC packet
      Fee                 Fee
      // the source port unique identifier
      SourcePortId        string
      // the source channel unique identifer
      SourceChannelId     string
      // account address to refund fee if necessary
      Signer              string
      // optional list of relayers permitted to the receive packet fee
      Relayers            []string
    }
    ```

    The `Fee` message contained in this synchronous fee payment method configures different fees which will be paid out for `MsgRecvPacket`, `MsgAcknowledgement`, and `MsgTimeout`/`MsgTimeoutOnClose`. 

    ```
    type Fee struct {
	    RecvFee             types.Coins
	    AckFee              types.Coins
	    TimeoutFee          types.Coin`
    }
    ```

2. `MsgPayPacketFeeAsync`, which enables the asynchronous escrowing of fees for a specified packet:

    Note that a packet can be 'topped up' multiple times with additional fees of any coin denomination by broadcasting multiple `MsgPayPacketFeeAsync` messages.

    ```
    type MsgPayPacketFeeAsync struct {
      // unique packet identifier comprised of the channel ID, port ID and sequence
      PacketId            channeltypes.PacketId
      // the packet fee associated with a particular IBC packet
      PacketFee           PacketFee
    }
    ```

    where the `PacketFee` also specifies the `Fee` to be paid as well as the refund address for fees which are not paid out
    ```
    type PacketFee struct {
	    Fee                    Fee
	    RefundAddress          string
	    Relayers               []]string
    }
    ```

Please see our [wiki](https://github.com/cosmos/ibc-go/wiki/Fee-enabled-fungible-token-transfers) for example flows on how to use these messages to incentivise a token transfer channel using a CLI.

# Paying out the escrowed fees
    
In the case of a successful transaction, `RecvFee` will be paid out to the designated counterparty payee address which has been registered on the receiver chain and sent back with the `MsgAcknowledgement`, `AckFee` will be paid out to the relayer address which has submitted the `MsgAcknowledgement` on the sending chain (or the registered payee in case one has been registered for the relayer address), and `TimeoutFee` will be reimbursed to the account which escrowed the fee. In cases of timeout transactions, `RecvFee` and `AckFee` will be reimbursed. 

Please note that fee payments are built on the assumption that sender chains are the source of incentives — the chain that sends the packets is the same chain where fee payments will occur -- please see the [relayer operator section](../ics29-fee/fee-distribution.md) to understand the flow for registering payee and counterparty payee (fee receiving) addresses.

# A locked fee middleware module

The fee middleware module can become locked if the situation arises that the escrow account for the fees does not have sufficient funds to pay out the fees which have been escrowed for each packet. This situation indicates a severe bug. In this case, the fee module will be locked until manual intervention fixes the issue. 

A locked fee module will simply skip fee logic and continue on to the underlying packet flow. A channel with a locked fee module will temporarily function as a fee disabled channel, and the locking of a fee module will not affect the continued flow of packets over the channel.
