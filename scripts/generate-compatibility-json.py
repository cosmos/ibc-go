#!/usr/bin/python3

import argparse
import json
import re
from typing import List, Dict

import requests
import semver

COMPATIBILITY_FLAG = "compatibility"
FROM_VERSION = "from_version"
# FROM_VERSIONS should be specified on individual tests if the features under test are only supported
# from specific versions of release lines.
FROM_VERSIONS = "from_versions"
# SKIP is a flag that can be used to skip a test from running in compatibility tests.
SKIP = "skip"
# fields will contain arbitrary key value pairs in comments that use the compatibility flag.
FIELDS = "fields"
TEST_SUITE = "test_suite"
TESTS = "tests"
RELEASES_URL = "https://api.github.com/repos/cosmos/ibc-go/releases"
# CHAIN_A should be specified if just chain-a -> chain-b tests should be run.
CHAIN_A = "chain-a"
# CHAIN_B should be specified if just chain-b -> chain-a tests should be run.
CHAIN_B = "chain-b"
# ALL is the default value chosen, and used to indicate that a test matrix which contains.
# both chain-a -> chain-b and chain-b -> chain-a tests should be run.
ALL = "all"
HERMES = "hermes"
DEFAULT_IMAGE = "ghcr.io/cosmos/ibc-go-simd"
RLY = "rly"
# MAX_VERSION is a version string that will be greater than any other semver version.
MAX_VERSION = "9999.999.999"
RELEASE_PREFIX = "release-"


def parse_version(version: str) -> semver.Version:
    """
    parse_version takes in a version string which can be in multiple formats,
    and converts it into a valid semver.Version which can be compared with each other.
    The version string is a docker tag. It can be in the format of
    - main
    - v1.2.3
    - 1.2.3
    - release-v1.2.3 (a tagged release)
    - release-v1.2.x (a release branch)
    """
    if version.startswith(RELEASE_PREFIX):
        # strip off the release prefix and parse the actual version
        version = version[len(RELEASE_PREFIX):]
    if version.startswith("v"):
        # semver versions do not include a "v" prefix.
        version = version[1:]
    # ensure "main" is always greater than other versions for semver comparison.
    if version == "main":
        # main will always be the newest release.
        version = MAX_VERSION
    if version.endswith("x"):
        # we always assume the release branch is newer than the previous release.
        # for example, release-v9.0.x is newer than release-v9.0.1
        version = version.replace("x", "999", 1)
    return semver.Version.parse(version)


def parse_args() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(description="Generate Compatibility JSON.")
    parser.add_argument(
        "--file",
        help="The test file to look at. Specify the path to a file under e2e/tests",
    )
    parser.add_argument(
        "--release-version",
        default="main",
        help="The version to run tests for.",
    )
    parser.add_argument(
        "--image",
        default=DEFAULT_IMAGE,
        help=f"Specify the image to be used in the test. Default: {DEFAULT_IMAGE}",
    )
    parser.add_argument(
        "--relayer",
        choices=[HERMES, RLY],
        default=HERMES,
        help=f"Specify relayer, either {HERMES} or {RLY}",
    )
    parser.add_argument(
        "--chain",
        choices=[CHAIN_A, CHAIN_B, ALL],
        default=ALL,
        help=f"Specify the chain to run tests for must be one of ({CHAIN_A}, {CHAIN_B}, {ALL})",
    )
    return parser.parse_args()


def main():
    args = parse_args()

    file_metadata = _build_file_metadata(args.file)
    tags = _get_ibc_go_releases(args.release_version)

    # extract the "from_version" annotation specified in the test file.
    # this will be the default minimum version that tests will use.
    min_version = parse_version(file_metadata[FIELDS][FROM_VERSION])

    all_versions = [parse_version(v) for v in tags]

    # get all tags between the min and max versions.
    tags_to_test = _get_tags_to_test(min_version, all_versions)

    # we also want to test the release version against itself, as well as already released versions.
    tags_to_test.append(args.release_version)

    # for each compatibility test run, we are using a single test suite.
    test_suite = file_metadata[TEST_SUITE]

    # all possible test files that exist within the suite.
    test_functions = file_metadata[TESTS]

    include_entries = []

    seen = set()
    for test in test_functions:
        for version in tags_to_test:
            if not _test_should_be_run(test, version, file_metadata[FIELDS]):
                continue

            _add_test_entries(include_entries, seen, version, test_suite, test, args)

    # compatibility_json is the json object that will be used as the input to a github workflow
    # which will expand out into a matrix of tests to run.
    compatibility_json = {
        "include": include_entries,
    }
    _validate(compatibility_json)

    # output the json on a single line. This ensures the output is directly passable to a github workflow.
    print(json.dumps(compatibility_json), end="")


def _add_test_entries(include_entries, seen, version, test_suite, test, args):
    """_add_test_entries adds two different test entries to the test_entries list. One for chain-a -> chain-b and one
    from chain-b -> chain-a. entries are only added if there are no duplicate entries that have already been added."""

    # add entry from chain-a -> chain-b
    _add_test_entry(include_entries, seen, args.chain, CHAIN_A, args.release_version, version, test_suite, test,
                    args.relayer, args.image)
    # add entry from chain-b -> chain-a
    _add_test_entry(include_entries, seen, args.chain, CHAIN_B, version, args.release_version, test_suite, test,
                    args.relayer, args.image)


def _add_test_entry(include_entries, seen, chain_arg, chain, version_a="", version_b="", entrypoint="", test="",
                    relayer="",
                    chain_image=""):
    """_add_test_entry adds a test entry to the include_entries list if it has not already been added."""
    entry = (version_a, version_b, test, entrypoint, relayer, chain_image)
    # ensure we don't add duplicate entries.
    if entry not in seen and chain_arg in (chain, ALL):
        include_entries.append(
            {
                "chain-a": version_a,
                "chain-b": version_b,
                "entrypoint": entrypoint,
                "test": test,
                "relayer-type": relayer,
                "chain-image": chain_image
            }
        )
        seen.add(entry)


def _get_tags_to_test(min_version: semver.Version, all_versions: List[semver.Version]):
    """return all tags that are between the min and max versions"""
    max_version = max(all_versions)
    return ["v" + str(v) for v in all_versions if min_version <= v <= max_version]


def _validate(compatibility_json: Dict):
    """validates that the generated compatibility json fields will be valid for a github workflow."""
    if "include" not in compatibility_json:
        raise ValueError("no include entries found")

    required_keys = frozenset({"chain-a", "chain-b", "entrypoint", "test", "relayer-type", "chain-image"})
    for k in required_keys:
        for item in compatibility_json["include"]:
            if k not in item:
                raise ValueError(f"key {k} not found in {item.keys()}")
            if not item[k]:
                raise ValueError(f"key {k} must have non empty value")

    if len(compatibility_json["include"]) > 256:
        # if this error occurs, split out the workflow into two jobs, one for chain-a and one for chain-b
        # using the --chain flag for this script.
        raise ValueError(f"maximum number of jobs exceeded (256): {len(compatibility_json['include'])}. "
                         f"Consider using the --chain argument to split the jobs.")


def _test_should_be_run(test_name: str, version: str, file_fields: Dict) -> bool:
    """determines if the test should be run. Each test can have its own versions defined, if it has been defined
    we can check to see if this test should run, based on the other test parameters.

    If no custom version is specified, the test suite level version is used to determine if the test should run.
    """

    # the test has been explicitly marked to be skipped for compatibility tests.
    if file_fields.get(f"{test_name}:{SKIP}") == "true":
        return False

    test_semver_version = parse_version(version)

    specified_from_version = file_fields.get(f"{test_name}:{FROM_VERSION}")
    if specified_from_version is not None:
        # the test has specified a minimum version for which to run.
        return test_semver_version >= parse_version(specified_from_version)

    # check to see if there is a list of versions that this test should run for.
    specified_versions_str = file_fields.get(f"{test_name}:{FROM_VERSIONS}")

    # no custom minimum version defined for this test
    # run it as normal using the from_version specified on the test suite.
    if specified_versions_str is None:
        # if there is nothing specified for this particular test, we just compare it to the version
        # specified at the test suite level.
        test_suite_level_version = file_fields[FROM_VERSION]
        return test_semver_version >= parse_version(test_suite_level_version)

    specified_versions = specified_versions_str.split(",")

    for v in specified_versions:
        semver_v = parse_version(v)
        # if the major and minor versions match, there was a specified release line for this version.
        # do a comparison on that version to determine if the test should run.
        if semver_v.major == test_semver_version.major and semver_v.minor == test_semver_version.minor:
            return semver_v >= test_semver_version

    # there was no version defined for this version's release line, but there were versions specified for other release
    # lines, we assume we should not be running the test.
    return False


def _get_ibc_go_releases(from_version: str) -> List[str]:
    releases = []

    from_version_semver = parse_version(from_version)

    # ref: documentation https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28#list-releases
    resp = requests.get(RELEASES_URL, params={"per_page": 1000})
    resp.raise_for_status()

    response_body = resp.json()

    all_tags = [release["tag_name"] for release in response_body]
    for tag in all_tags:
        # skip alphas, betas and rcs
        if any(t in tag for t in ("beta", "rc", "alpha", "icq")):
            continue
        try:
            semver_tag = parse_version(tag)
        except ValueError:  # skip any non semver tags.
            continue

        # get all versions
        if semver_tag <= from_version_semver:
            releases.append(tag)

    return releases


def _build_file_metadata(file_name: str) -> Dict:
    """_build_file_metadata constructs a dictionary of metadata from the test file."""
    file_lines = _load_file_lines(file_name)
    return {
        TEST_SUITE: _extract_test_suite_function(file_lines),
        TESTS: _extract_all_test_functions(file_lines),
        FIELDS: _extract_script_fields(file_lines)
    }


def _extract_test_suite_function(file_lines: List[str]) -> str:
    """extracts the name of the test suite function in the file. It is assumed
    there is exactly one test suite defined"""
    for line in file_lines:
        line = line.strip()
        if "(t *testing.T)" in line:
            return re.search(r"func\s+(.*)\(", line).group(1)
    raise ValueError("unable to find test suite in file lines")


def _extract_all_test_functions(file_lines: List[str]) -> List[str]:
    """creates a list of all test functions that should be run in the compatibility tests
     based on the version provided"""
    all_tests = []
    for i, line in enumerate(file_lines):
        line = line.strip()

        if line.startswith("//"):
            continue

        if not _is_test_function(line):
            continue

        test_function = _test_function_match(line).group(1)
        all_tests.append(test_function)

    return all_tests


def _is_test_function(line: str) -> bool:
    """determines if the line contains a test function definition."""
    return _test_function_match(line) is not None


def _test_function_match(line: str) -> re.Match:
    return re.match(r".*\).*(Test.*)\(\)", line)


def _extract_script_fields(file_lines: List[str]) -> Dict:
    """extract any field in the format of
    // compatibility:field_name:value
    e.g.
    // compatibility:from_version: v7.0.0
    // compatibility:foo: bar
    becomes
    {
      "from_version": "v7.0.0",
      "foo": "bar"
    }
    """
    script_fields = {}
    for line in file_lines:
        line = line.strip()
        match = re.match(rf"//\s*{COMPATIBILITY_FLAG}\s*:\s*(.*):\s*(.*)", line)
        if match:
            script_fields[match.group(1)] = match.group(2)
    return script_fields


def _load_file_lines(file_name: str) -> List[str]:
    with open(file_name, "r") as f:
        return f.readlines()


if __name__ == "__main__":
    main()
