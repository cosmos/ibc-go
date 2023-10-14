#!/usr/bin/python3

import argparse
import json
import re
import requests
import semver
from typing import List, Dict

COMPATIBILITY_FLAG = "compatibility"
FROM_VERSION = "from_version"
RELEASES_URL = "https://api.github.com/repos/cosmos/ibc-go/releases"
CHAIN_A = "chain-a"
CHAIN_B = "chain-b"
HERMES = "hermes"
RLY = "rly"

def parse_args() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(description="Generate Compatibility JSON.")
    parser.add_argument(
        "--file",
        help="The test file to look at.",
    )
    parser.add_argument(
        "--version",
        help="The version to run tests for.",
    )
    parser.add_argument(
        "--release-version",
        help="The version to run tests for.",
    )
    parser.add_argument(
        "--chain",
        choices=[CHAIN_A, CHAIN_B],
        default=CHAIN_A,
        help=f"Specify {CHAIN_A} or {CHAIN_B} for use with the json files.",
    )
    parser.add_argument(
        "--relayer",
        choices=[HERMES, RLY],
        default=HERMES,
        help=f"Specify relayer, either {HERMES} or {RLY}",
    )

    return parser.parse_args()


def _to_release_version_str(version: str) -> str:
    """convert a version to the release version tag
    e.g.
    v4.4.0 -> releases-v4.4.x
    v7.3.0 -> releases-v7.3.x
    """
    return "".join(("release-", version[0:len(version) - 1], "x"))


def _from_release_tag_to_regular_tag(release_version: str) -> str:
    """convert a version to the release version tag
    e.g.
    releases-v4.4.x -> v4.4.0
    releases-v7.3.x -> v7.3.0
    """
    return "".join((release_version[len("release-"):len(release_version) - 1], "0"))


def _get_max_version(versions):
    """remove the v as it's not a valid semver version"""
    return max([v for v in versions])


def _get_tags_to_test(min_version: semver.Version, max_version: semver.Version, all_versions: List[semver.Version]):
    """return all tags that are between the min and max versions"""
    return [v for v in all_versions if min_version < v < max_version]

def main():
    args = parse_args()
    extracted_version = _from_release_tag_to_regular_tag(args.release_version)
    file_lines = _load_file_lines(args.file)
    file_metadata = _build_file_metadata(file_lines)
    tags = _get_ibc_go_releases(extracted_version)


    min_version = file_metadata["from_version"]
    max_version = _get_max_version(tags)

    release_versions = [args.release_version]

    tags_to_test = _get_tags_to_test(min_version, max_version, tags)

    other_versions = tags_to_test

    # if we are specifying chain B, we invert the versions.
    if args.chain == CHAIN_B:
        release_versions, other_versions = other_versions, release_versions

    test_suite = file_metadata["test_suite"]
    test_functions = file_metadata["tests"]
    # print(test_functions)
    compatibility_json = {
        "chain-a": release_versions,
        "chain-b": list(map(_semver_to_str, other_versions)) + release_versions, # TODO: clean this up, it's a bit hacky
        "entrypoint": [test_suite],
        "test": test_functions,
        "relayer-type": [args.relayer]
    }

    _validate(compatibility_json, "")

    # output the json on a single line. This ensures the output is directly passable to a github workflow.
    print(json.dumps(compatibility_json), end="")


def _validate(compatibility_json: Dict, version: str):
    """validates that the generated compatibility json fields will be valid for a github workflow."""
    required_keys = frozenset({"chain-a", "chain-b", "entrypoint", "test", "relayer-type"})
    for k in required_keys:
        if k not in compatibility_json:
            raise ValueError(f"key {k} not found in {compatibility_json.keys()}")

    if compatibility_json["chain-a"] == compatibility_json["chain-b"]:
        raise ValueError("chain ids must be different")

    if len(compatibility_json["entrypoint"]) != 1:
        raise ValueError(f"found more than one entrypoint: {compatibility_json['entrypoint']}")

    if len(compatibility_json["test"]) <= 0:
        raise ValueError(f"no tests found for version {version}")

    if len(compatibility_json["relayer-type"]) <= 0:
        raise ValueError("no relayer specified")


def _get_ibc_go_releases(from_version: str) -> List[semver.Version]:
    releases = []

    from_version_semver = _to_semver(from_version)

    resp = requests.get(RELEASES_URL)
    resp.raise_for_status()

    response_body = resp.json()

    all_tags = [release["tag_name"] for release in response_body]
    for tag in all_tags:
        # skip alphas, betas and rcs
        if any(c in tag for c in ("beta", "rc", "alpha", "icq")):
            continue
        try:
            semver_tag = _str_to_semver(tag)
        except ValueError:  # skip any non semver tags.
            continue

        # get all versions
        if from_version_semver >= semver_tag:
            releases.append(semver_tag)

    return releases


def _extract_all_test_functions(file_lines: List[str]) -> List[str]:
    """creates a list of all test functions that should be run in the compatibility tests based on the version provided"""
    all_tests = []
    for i, line in enumerate(file_lines):
        line = line.strip()

        # TODO: handle block comments
        if line.startswith("//"):
            continue

        if not _is_test_function(line):
            continue

        test_function = _test_function_match(line).group(1)
        all_tests.append(test_function)

    return all_tests


# def _extract_function_name_from_line(line: str) -> str:
#     """extract the name of the go test function from the line of source code provided."""
#     return re.search(r".*(Test.*)\(\)", line).group(1)


def _test_function_match(line: str) -> re.Match:
    return re.match(r".*(Test.*)\(\)", line)


def _is_test_function(line: str) -> bool:
    """determines if the line contains a test function definition."""
    return _test_function_match(line) is not None


def _extract_test_suite_function(file_lines: List[str]) -> str:
    """extracts the name of the test suite function in the file. It is assumed there is exactly one test suite defined"""
    for line in file_lines:
        line = line.strip()
        if "(t *testing.T)" in line:
            return re.search(r"func\s+(.*)\(", line).group(1)
    raise ValueError("unable to find test suite in file lines")


def _load_file_lines(file_name: str) -> List[str]:
    with open(file_name, "r") as f:
        return f.readlines()


def _extract_from_version(file_lines: List[str]) -> str:
    for line in file_lines:
        line = line.strip()
        match = re.match(rf"//\s*{COMPATIBILITY_FLAG}:{FROM_VERSION}.*v(.*)", line)
        if match:
            return semver.Version.parse(match.group(1))
    raise ValueError("no from version found in file")


def _semver_to_str(semver_version: semver.Version) -> str:
    return f"v{semver_version.major}.{semver_version.minor}.{semver_version.patch}"

def _str_to_semver(str_version: str) -> semver.Version:
    if str_version.startswith("v"):
        str_version = str_version[1:]
    return semver.Version.parse(str_version)

def _build_file_metadata(file_lines: List[str]) -> Dict:
    return {
        "test_suite": _extract_test_suite_function(file_lines),
        "tests": _extract_all_test_functions(file_lines),
        "from_version": _extract_from_version(file_lines)
    }


if __name__ == "__main__":
    main()
