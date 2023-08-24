# Callbacks Testing SimApp

This testing directory is a duplicate of the ibc-go testing directory.
It is only here as a way of creating a separate SimApp binary to avoid introducing a dependency on the callbacks
module from within ibc-go.

The simapp can be built with the workflow found [here](../../../../../.github/workflows/build-callbacks-simd-image-from-tag.yml).
