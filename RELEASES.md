# Releases

IBC-Go follows [semantic versioning](https://semver.org), but with the following deviations:

- A state-machine breaking change will result in an increase of the minor version Y (x.Y.z | x > 0).
- An API breaking change will result in an increase of the major number (X.y.z | x > 0). Please note that these changes **will be backwards compatible** (as opposed to canonical semantic versioning; read [Backwards compatibility](#backwards) for a detailed explanation).

This is visually explained in the following decision tree:

<p align="center">
  <img src="releases-decision-tree.png?raw=true" alt="Releases decision tree" width="40%" />
</p>

When bumping the dependencies of [Cosmos SDK](https://github.com/cosmos/cosmos-sdk) and [Tendermint](https://github.com/tendermint/tendermint) we will only treat patch releases as non state-machine breaking.

## <a name="backwards"></a> Backwards compatibility

[ibc-go](https://github.com/cosmos/ibc-go) and the [IBC protocol specification](https://github.com/cosmos/ibc) maintain different versions. Furthermore, ibc-go serves several different user groups (chains, IBC app developers, relayers, IBC light client developers). Each of these groups has different expectations of what *backwards compatible* means. It simply isn't possible to categorize a change as backwards or non backwards compatible for all user groups. We are primarily interested in when our API breaks and when changes are state machine breaking (thus requiring a coordinated upgrade). This is scoping the meaning of ibc-go to that of those interacting with the code (IBC app developers, relayers, IBC light client developers), not chains using IBC to communicate (that should be encapsulated by the IBC protocol specification versioning).

To summarize: **All our ibc-go releases allow chains to communicate successfully with any chain running any version of our code**. That is to say, we are still using IBC protocol specification v1.0. 

We ensure all major releases are supported by relayers ([hermes](https://github.com/informalsystems/ibc-rs), [rly](https://github.com/strangelove-ventures/relayer) and [ts-relayer](https://github.com/confio/ts-relayer) at the moment) which can relay between the new major release and older releases. We have no plans of upgrading to an IBC protocol specification v2.0, as this would be very disruptive to the ecosystem.

## Release cycle

IBC-Go follows a traditional release cycle involving an alpha, beta, and rc (release candidate) releases before finalizing a new version. As ibc-go works in a non-traditional area, we apply our own interpretation to each release type. We reserve the right to make both go API breaking changes and state machine breaking changes throughout the entire release cycle. The stable release guarentees do not go into affect until a final release is performed. 

It is never advisable to use a non-final release in production. 

### Alpha

Alpha releases are intended to make available new features as soon as they are functional. No correctness guarentees are made and alpha releases **may** contain serious security vulnerabilities, bugs, and lack of user tooling, so long as they don't affect the core functionality. 

Initial users of alpha releases are expected to be advanced, patient, and capable of handling unusual errors. Very basic integration testing will be performed by the ibc-go development team before alpha releases.  

An internal audit is typically performed before the alpha release allowing the development team to guage the maturity and stability of changes included in the next release. 

### Beta

Beta releases are intended to signal design stability. While the go API is still subject to change, the core design of the new features should not be. Developers integrating the new features should expect to handle breaking changes when upgrading to RC's. 

Beta releases should not be made with known bugs or security vulnerabilities. Beta releases should focus on ironing out remaining bugs and filling out the UX functionality required by a final release. Beta releases should have a clearly defined scope of the features that will be included in the release. Only highly requested feature additions should be acted upon in this phase. 

When the development team has determined a release is ready to enter the RC phase, a final security audit should be performed. The security audit should be limited to looking for bugs and security vulnerabilities. Code improvements may be noted, but they should not be acted upon unless highly desirable. 

### RC 

RC's are release candidates. Final releases should contain little to no changes in comparison to the latest RC. Changes included in between RC releases should be limited to:
- Improved testing
- UX additions
- Bug fixes
- Highly requested changes by the community

A release should not be finalized until the development team and the external community have done sufficient integration tests on the targeted release. 

## Stable Release Policy

The beginning of a new major release series is marked by the release of a new major version. A major release series is comprised of all minor and patch releases made under the same major version number. The series continues to receive bug fixes (released as minor or patch releases) until it reaches end of life. The date when a major release series reaches end of life is determined by one of the two following methods:
- If the next major release is made within the first 6 months, then the end of life date of the major release series is 1 year after its initial release. 
- If the next major release is made 6 months after the initial release, then the end of life date of the major release series is 6 months after the release date of the next major release.

For example, if the current major release series is v1 and was released on January 1st, 2022, then v1 will be supported at least until January 1st, 2023. If v2 is published on August 1st 2022, then v1's end of life will be March 1st, 2023. 

Only the following major release series have a stable release status:

|Release|End of Life Date|
|-------|-------|
|`v1.3.x`|July 01, 2022|
|`v1.4.x`|July 01, 2022|
|`v2.1.x`|February 01, 2023|
|`v2.2.x`|February 01, 2023|
|`v3.0.x`|March 15, 2023|

**Note**: The v1 major release series will reach end of life 6 months after merging this policy. v2 will reach end of life one year after merging this policy. 

### What pull requests will be included in stable patch-releases?

Pull requests that fix bugs and add features that fall in the following categories:

* **Severe regressions**.
* Bugs that may cause **client applications** to be **largely unusable**.
* Bugs that may cause **state corruption or data loss**.
* Bugs that may directly or indirectly cause a **security vulnerability**.
* Non-breaking features that are strongly requested by the community.
* Non-breaking CLI improvements that are strongly requested by the community.

### What pull requests will NOT be automatically included in stable patch-releases?

As rule of thumb, the following changes will **NOT** be automatically accepted into stable point-releases:

* **State machine changes**, unless the previous behaviour would result in a consensus halt.
* **Protobuf-breaking changes**.
* **Client-breaking changes**, i.e. changes that prevent gRPC, HTTP and RPC clients to continue interacting with the node without any change.
* **API-breaking changes**, i.e. changes that prevent client applications to *build without modifications* to the client application's source code.
* **CLI-breaking changes**, i.e. changes that require usage changes for CLI users.

## Graphics

The decision tree above was generated with the following code:

```
%%{init: 
    {'theme': 'default',
     'themeVariables': 
        {'fontFamily': 'verdana', 'fontSize': '13px'}
    }
}%%
flowchart TD
    A(Change):::c --> B{API breaking?}
    B:::c --> |Yes| C(Increase major version):::c
    B:::c --> |No| D{state-machine breaking?}
    D:::c --> |Yes| G(Increase minor version):::c
    D:::c --> |No| H(Increase patch version):::c
    classDef c fill:#eee,stroke:#aaa
```

using [Mermaid](https://mermaid-js.github.io)'s [live editor](https://mermaid.live).
