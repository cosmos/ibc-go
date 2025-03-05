---
title: Protobuf Documentation
sidebar_label: Protobuf Documentation
sidebar_position: 9
slug: /ibc/proto-docs
---


# Protobuf Documentation

## Buf Schema Registry

IBC-Go Protobuf definitions are hosted on the [Buf Schema Registry](https://buf.build/cosmos/ibc).

The registry includes all IBC-Go Protobuf definitions and is updated:
- When changes are made to Protobuf files on the `main` branch
- When a new version of IBC-Go is released with a tag (e.g., `v7.3.0`)

You can browse the Protobuf definitions directly on the Buf Schema Registry:
- [Main branch](https://buf.build/cosmos/ibc/docs/main)
- Tagged versions are available in the "Tags" section

## Using Tagged Versions in Your Projects

When depending on IBC-Go Protobuf definitions in your own projects, you can specify a particular tagged version to ensure API stability:

```yaml
# In your buf.yaml file
deps:
  - buf.build/cosmos/ibc:v7.3.0  # Replace with your desired version
```

This ensures your project always uses the same version of the API definitions, regardless of updates to the main branch.

## Generated Documentation

For a complete reference of all IBC Protobuf definitions, see the [generated Protobuf documentation](https://buf.build/cosmos/ibc/docs/main).
