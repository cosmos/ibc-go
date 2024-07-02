---
title: Run 2 Cosmos SDK Blockchains Locally
sidebar_label: Run 2 Cosmos SDK Blockchains Locally
sidebar_position: 3
slug: /channel-upgrades/run-chains
---

# Run 2 Cosmos SDK blockchains locally

The gm tool uses a [configuration file](https://github.com/informalsystems/gm/blob/master/gm.toml). This tutorial uses the following configuration file for gm:

```yaml title="gm.toml"
[global]
add_to_hermes=true
home_dir="~/testing/gm"

[global.hermes]
binary="~/testing/hermes/hermes"
config="~/testing/hermes/config.toml"

[chain1]
  gaiad_binary="~/testing/bin/chain1/simd"  
  ports_start_at=27000

[chain2]
  gaiad_binary="~/testing/bin/chain2/simd" 
  ports_start_at=27010
```

The configuration file needs to be placed in `$HOME/.gm`. This configuration file sets up 2 blockchains (`chain1` and `chain2`), each with 2 accounts (1 validator, 1 wallet). The ports where the CometBFT RPC interface for each chain is 27000 for `chain1` and 27010 for `chain2`.

In order to shorten the voting period of governance proposal, we are going to change some of the `x/gov` module parameters in the `genesis.json` file, so that we can complete the upgrade faster. These are the changes needed in the `genesis.json` of `chain1`:

```json title="genesis.json"
"gov": {
  "starting_proposal_id": "1",
  "deposits": [],
  "votes": [],
  "proposals": [],
  "deposit_params": null,
  "voting_params": null,
  "tally_params": null,
  "params": {
    "min_deposit": [
      {
        "denom": "stake",
// minus-diff-line
-       "amount": "10000000"
// plus-diff-line
+       "amount": "100"
      }
    ],
    "max_deposit_period": "172800s",
// minus-diff-line
-   "voting_period": "172800s",
// plus-diff-line
+   "voting_period": "180s",
    "quorum": "0.334000000000000000",
// minus-diff-line
-   "threshold": "0.500000000000000000",
// plus-diff-line
+   "threshold": "0.300000000000000000",
    "veto_threshold": "0.334000000000000000",
    "min_initial_deposit_ratio": "0.000000000000000000",
    "proposal_cancel_ratio": "0.500000000000000000",
    "proposal_cancel_dest": "",
    "expedited_voting_period": "86400s",
    "expedited_threshold": "0.667000000000000000",
    "expedited_min_deposit": [
      {
        "denom": "stake",
        "amount": "50000000"
      }
    ],
    "burn_vote_quorum": false,
    "burn_proposal_deposit_prevote": false,
    "burn_vote_veto": true,
    "min_deposit_ratio": "0.010000000000000000"
  },
  "constitution": ""
}
```

We start both blockchains by running the following command:

```bash
gm start
```

For convenience, we are going to store a few account addresses as variables in the current shell environment. Execute the following commands to store the relayer addresses on chains `chain1` and `chain2`, respectively:

```bash
export RLY_CHAIN1=$(simd keys show wallet -a \
--keyring-backend test \
--home ../../gm/chain1) && echo $RLY_CHAIN1;
export RLY_CHAIN2=$(simd keys show wallet -a \
--keyring-backend test \
--home ../../gm/chain2) && echo $RLY_CHAIN2;
```

And execute also the following commands to store the validator account addresses on chains `chain1` and `chain2` that we will use throughout this tutorial:

```bash
export VALIDATOR_CHAIN1=$(simd keys show validator -a \
--keyring-backend test \
--home ../../gm/chain1) && echo $VALIDATOR_CHAIN1;
export VALIDATOR_CHAIN2=$(simd keys show validator -a \
--keyring-backend test \
--home ../../gm/chain2) && echo $VALIDATOR_CHAIN2;
```
