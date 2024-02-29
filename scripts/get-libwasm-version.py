#!/usr/bin/env python3
"""
The purpose of this script is to output the version of the libwasm library that is
specified in the go.mod file in the wasm module.

This should be passed as a build argument to the Dockerfile to determine which static library should
be added.

usage: get-libwasm-version.py [-h] [--get-version | --no-get-version | --get-checksum | --no-get-checksum] [--wasm-library WASM_LIBRARY] [--wasm-go-mod-path WASM_GO_MOD_PATH]

Wasm dockerfile utility

options:
  -h, --help            show this help message and exit
  --get-version, --no-get-version
                        Get the current version of CosmWasm specified in wasm module.
  --get-checksum, --no-get-checksum
                        Returns the checksum of the libwasm library for the provided version.
  --wasm-library WASM_LIBRARY
                        The name of the library to return the checksum for.
  --wasm-go-mod-path WASM_GO_MOD_PATH
                        The relative path to the go.mod file for the wasm module.
"""

import argparse
import requests

WASM_IMPORT = "github.com/CosmWasm/wasmvm"


def _get_wasm_version(wasm_go_mod_path: str) -> str:
    """get the version of the cosm wasm module from the go.mod file"""
    with open(wasm_go_mod_path, "r") as f:
        for line in f:
            if WASM_IMPORT in line:
                return _extract_wasm_version(line)
    raise ValueError(f"Could not find {WASM_IMPORT} in {wasm_go_mod_path}")


def _get_wasm_lib_checksum(wasm_version: str, wasm_lib: str) -> str:
    """get the checksum of the wasm library for the given version"""
    checksums_url = f"https://github.com/CosmWasm/wasmvm/releases/download/{wasm_version}/checksums.txt"
    resp = requests.get(checksums_url)
    resp.raise_for_status()

    for line in resp.text.splitlines():
        if wasm_lib in line:
            return line.split(" ")[0].strip()

    raise ValueError(f"Could not find {wasm_lib} in {checksums_url}")


def _extract_wasm_version(line: str) -> str:
    """extract the version from a line in the go.mod file"""
    return line.split(" ")[1].strip()


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Wasm dockerfile utility")

    group = parser.add_mutually_exclusive_group()
    group.add_argument(
        "--get-version",
        action=argparse.BooleanOptionalAction,
        help="Get the current version of CosmWasm specified in wasm module.",
    )
    group.add_argument(
        "--get-checksum",
        action=argparse.BooleanOptionalAction,
        help="Returns the checksum of the libwasm library for the provided version."
    )
    parser.add_argument(
        "--wasm-library",
        default="libwasmvm_muslc.x86_64.a",
        help="The name of the library to return the checksum for."
    )
    parser.add_argument(
        "--wasm-go-mod-path",
        default="modules/light-clients/08-wasm/go.mod",
        help="The relative path to the go.mod file for the wasm module."
    )
    return parser.parse_args()


def main(args: argparse.Namespace):
    if args.get_version:
        version = _get_wasm_version(args.wasm_go_mod_path)
        print(version)
        return
    if args.get_checksum:
        checksum = _get_wasm_lib_checksum(_get_wasm_version(args.wasm_go_mod_path), args.wasm_library)
        print(checksum)
        return


if __name__ == "__main__":
    main(_parse_args())
