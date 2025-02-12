# Compatibility Generation

## Introduction

The generate-compatibility-json.py script is used to generate matrices that can be fed into github workflows
as the matrix for the compatibility job.

This is done by generating a matrix of all possible combinations based on a provided release branch
e.g. release-v10.0.x

## Matrix Generation

The generation script is provided a file containing tests, e.g. e2e/tests/transfer/base_test.go and a version under
test. The script will then look at any annotations present in the test in order to determine which tests should
and shouldn't be run.

## Annotations

Annotations can be arbitrarily added to the test files in order to control which tests are run.

The general syntax is:

`//compatibility:{some_annotation}:{value}`

In order to apply an annotation to a specific test, the following syntax is used:

`//compatibility:{TEST_NAME}:{annotation}:{value}`

The annotations can be present anywhere in the file, typically it is easiest to place the annotations near the test
or test suite they are controlling.

The following annotations are supported:

| Annotation              | Example Value        | Purpose                                                                                                                                                                                                                                                                                                             | Example in test file                                                                 |
|-------------------------|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------|
| from_version            | v7.4.0               | Tests should only run if a semver comparison is greater than or equal to this version. Generally this will just be the minimum supported version of ibc-go                                                                                                                                                          | // compatibility:from_version:v7.4.0                                                 |
| TEST_NAME:from_versions | v8.4.0,v8.5.0,v10.0.0 | For some tests, they should only be run against a specific release line. This annotation is test case specific, and ensures the test case is run based on the major and minor versions specified. If a version is provided to the tool, and a matching major minor version is not listed, the test will be skipped. | // compatibility:TestScheduleIBCUpgrade_Succeeds:from_versions: v8.4.0,v8.5.0,v10.0.0 |
| TEST_NAME:skip          | true                 | A flag to ensure that this test is not included in the compatibility tests at all.                                                                                                                                                                                                                                  | // compatibility:TestMsgSendTx_SuccessfulSubmitGovProposal:skip:true                 |

> Note: if additional control is required, the script can be modified to support additional annotations.
