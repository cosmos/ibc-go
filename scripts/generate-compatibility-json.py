#!/usr/bin/python3

import argparse
import json
import re
import requests
import semver
from typing import List, Dict

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



def main():
    args = parse_args()
    extracted_version = _from_release_tag_to_regular_tag(args.release_version)
    file_lines = _load_file_lines(args.file)
    test_suite_name = _extract_test_suite_function(file_lines)
    test_functions = _extract_test_functions_to_run(extracted_version, file_lines)

    # release_versions = [_to_release_version_str(args.version)]
    release_versions = [args.release_version]



    tags = _get_ibc_go_releases(extracted_version)
    tags.extend(release_versions)
    # print(args.version)
    # next_release_version = _from_release_tag_to_regular_tag(args.release_version)

    other_versions = tags

    # if we are specifying chain B, we invert the versions.
    if args.chain == CHAIN_B:
        release_versions, other_versions = other_versions, release_versions

    compatibility_json = {
        "chain-a": release_versions,
        "chain-b": other_versions,
        "entrypoint": [test_suite_name],
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


def _to_semver(version: str) -> semver.Version:
    if version.startswith("v"):
        version = version[1:]
    return semver.Version.parse(version)


def _get_ibc_go_releases(from_version: str) -> List[str]:
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
            semver_tag = _to_semver(tag)
        except ValueError:  # skip any non semver tags.
            continue
        if semver_tag >= from_version_semver:
            releases.append(tag)

    return releases


def _extract_test_functions_to_run(version: str, file_lines: List[str]) -> List[str]:
    """creates a list of all test functions that should be run in the compatibility tests based on the version provided"""
    test_function_names: List[str] = []
    for i, line in enumerate(file_lines):
        line = line.strip()

        if not line.startswith("//"):
            continue

        if FROM_VERSION in line:
            # TODO: do semver check instead of specific version match.
            if not re.match(fr"//\sfrom_version:\s({version})", line):
                continue

            # TODO: look for the name instead of assuming it's on the next line.
            idx_of_test_declaration = i + 1

            if idx_of_test_declaration >= len(file_lines):
                raise ValueError(
                    "index out of bounds, did not find a function associated with the 'from_version' annotation",
                )

            fn_name_line = file_lines[i + 1].strip()
            test_function_names.append(_extract_function_name_from_line(fn_name_line))

    return test_function_names


def _extract_function_name_from_line(line: str) -> str:
    """extract the name of the go test function from the line of source code provided."""
    return re.search(r".*(Test.*)\(\)", line).group(1)


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


if __name__ == "__main__":
    main()
