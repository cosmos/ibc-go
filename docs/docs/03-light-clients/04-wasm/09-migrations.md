---
title: Migrations
sidebar_label: Migrations
sidebar_position: 9
slug: /ibc/light-clients/wasm/migrations
---

# Migrations

This guide provides instructions for migrating 08-wasm versions.

## From ibc-go v7.3.x to ibc-go v8.0.x

## Chains

In the 08-wasm versions compatible with ibc-go v7.3.x and above from the v7 release line, the checksums of the uploaded Wasm bytecodes are all stored under a single key. From ibc-go v8.0.x the checksums are stored using [`collections.KeySet`](https://docs.cosmos.network/v0.50/build/packages/collections#keyset), whose full functionality became available in Cosmos SDK v0.50. There is therefore an [automatic migration handler](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/modules/light-clients/08-wasm/module.go#L115-L118) configured in the 08-wasm module to migrate the stored checksums to `collections.KeySet`.
