# Request For Maintainership

This document details the acceptance process for requests from external contributors who require for the ibc-go core team to take over **maintainership** of a complex block of code (ie: an application module or a light client implementation). It is the process we also follow internally for features that we develop which will go into the `ibc-go` codebase.

For projects who have created a custom IBC application and want us to list this application on the registry, please break out your module into its own repo for ease of import into chains!

- Create a repo with the custom module in a folder `x/` or `modules/`. 
- Please include an app that contains the custom module along with end-to-end tests that spins up the blockchain and runs the custom module. 
- See [cosmos/interchain-security](https://github.com/cosmos/interchain-security) for an example of this setup.

For contributors wishing to submit contributions to the codebase, please check our [Contributor Guidelines](https://github.com/cosmos/ibc-go/blob/main/CONTRIBUTING.md) :)


<p align="center">
  <img src="maintainership.png?raw=true" alt="maintainership" width="80%" />
</p>

## Step 1: Product check

Reach out to the IBC product team through susannah@interchain.io to coordinate use-case walkthrough.

Answer these questions in a requirements doc:

    What problem does this feature solve?

    What are the current solutions or workarounds? 

    Are there other versions or implementations?

    If there are other versions of this feature, why is this solution better?

    What are the use cases?

    Which users have confirmed they will use this? 

    How urgent is it to implement this feature?

    How soon after being developed would this be adopted? 

    What is the impact of this feature being adopted?

    Is there a specific need for this feature to be included in the `ibc-go` codebase, rather than in its own module repo?


Answers to these questions should also be detailed in a **discussion** in the `ibc-go` repo to open up the discussion to a wider audience, this can be done before or after the walkthrough. 

The acceptance criteria is based on the answers to these questions and the results of this product check, as well as of course an acceptable spec should the module be deemed to need one. Please see Step #2 below for the spec considerations.

In summary, the feature must solve a genuine problem, have users that would greatly benefit from the solution and be generic enough to benefit many users of the `ibc-go` implementation. 

## Step 2: Submit spec to the IBC protocol repo

A detailed review of the specification can be expected **within 2 weeks** of submission of the specification to the repo. Please notify the specification team if this does not occur, so it can be corrected as soon as possible. Please note that this timeline may be subject to amendment based on complexity of the spec and team capacity considering other ongoing reviews, but we will strive to ensure a 2 week turnaround.

Any IBC code that is expected to be implemented across different chains in order to function correctly must be specified in the IBC repo to be accepted unless exempted by the specification team. 
    
Unilateral software (ie. code that only needs to run on a single chain to be functional) need not be submitted. In these cases however, some sort of design document such as [ADR-008](https://github.com/cosmos/ibc-go/pull/1976/files) should be submitted.

If the associated module to be developed is expected to be submitted to the `ibc-go` team for maintainership, this should already be flagged at this step so that we can start thinking about/preparing our own capacity for the engineering team.

## Step 3: Prepare code for handover

*(this step can be initiated in parallel w/ spec submission, subject to feature complexity)*

Once the spec has been given initial approval, `ibc-go` engineering will coordinate a code walkthrough in preparation for taking the module into the repo. Any requested changes from the `ibc-go` engineering team after the code walkthrough should be discussed and/or addressed in a timely manner.

The code that is presented should adhere to our [code contributor guidelines](https://github.com/cosmos/ibc-go/blob/main/CONTRIBUTING.md).

More details on what code walkthrough should cover will be provided by the `ibc-go` engineering team on a case by case basis. However, the code should be sufficiently unit and [E2E tested](https://github.com/cosmos/ibc-go/blob/main/e2e/README.md). Think about preparing for this process similarly to submitting a codebase for audit :)

Please indicate the expected contribution of your team maintainership, if any. This contribution should also include ideas about devrels support and support for product on social media.

ETA for the actual handover will be subject to amendment based on feedback resulting from the code walkthrough.
