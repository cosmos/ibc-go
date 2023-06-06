#!/usr/bin/python3
import argparse
import os
import json
import enum
from typing import Tuple, Generator, Optional, Dict, Any

# JSON keys to search in.
KEYS: Tuple[str, str] = ("chain-a", "chain-b")
# Toggle if required. Indent = 4 matches jq.
DUMP_ARGS: Dict[Any, Any] = {
    "indent": 4,
    "sort_keys": False,
    "ensure_ascii": False,
}
# Suggestions for recent and new versions.
SUGGESTION: str = "for example, v4.3.0 or 4.3.0"
# Supported Operations
Operation = enum.Enum("Operation", ["ADD", "REMOVE", "REPLACE"])


def find_json_files(
    directory: str, ignores: Tuple[str] = (".",)
) -> Generator[str, None, None]:
    """Find JSON files in a directory. By default, ignore hidden directories."""
    for root, dirs, files in os.walk(directory):
        dirs[:] = (d for d in dirs if not d.startswith(ignores))
        for file_ in files:
            if file_.endswith(".json"):
                yield os.path.join(root, file_)


def has_release_version(json_file, keys: Tuple[str, str], version: str) -> bool:
    """Check if the json file has the version in question."""
    rows = (json_file[key] for key in keys)
    return any(version in row for row in rows)


def update_version(json_file, keys: Tuple[str, str], args: argparse.Namespace):
    """Update the versions as required in the json file."""
    recent, new, op = args.recent, args.new, args.type
    for row in (json_file[key] for key in keys):
        if recent not in row:
            continue
        if op == Operation.ADD:
            row.append(new)
            row.sort(reverse=True)
        else:
            index = row.index(recent)
            if op == Operation.REPLACE:
                row[index] = new
            elif op == Operation.REMOVE:
                del row[index]


def version_input(prompt: str, version: Optional[str]) -> str:
    """Input version if not supplied, make it start with a 'v' if it doesn't."""
    if version is None:
        version = input(prompt)
    return version if version.startswith(("v", "V")) else f"v{version}"


def require_version(args: argparse.Namespace):
    """Allow non-required version in argparse but request it if not provided."""
    args.recent = version_input(f"Recent version ({SUGGESTION}): ", args.recent)
    if args.type == Operation.REMOVE:
        return
    args.new = version_input(f"New version ({SUGGESTION}): ", args.new)


def parse_args() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(description="Update JSON files.")
    parser.add_argument(
        "--type",
        choices=[Operation.ADD.name, Operation.REPLACE.name, Operation.REMOVE.name],
        default=Operation.ADD,
        help="Type of version update: add a version, replace one or remove one.",
    )
    parser.add_argument(
        "--directory", default=".", help="Directory path where JSON files are located"
    )
    parser.add_argument(
        "--recent_version",
        dest="recent",
        help=f"Recent version to search in JSON files ({SUGGESTION})",
    )
    parser.add_argument(
        "--new_version",
        dest="new",
        help=f"New version to add in JSON files ({SUGGESTION})",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Allow for verbose output",
        default=False,
    )

    args = parser.parse_args()
    require_version(args)
    return args


def main(args: argparse.Namespace):
    for file_ in find_json_files(args.directory):
        with open(file_, "r+") as fp:
            json_file = json.load(fp)
            if not has_release_version(json_file, KEYS, args.recent):
                continue
            update_version(json_file, KEYS, args)
            fp.seek(0)
            json.dump(json_file, fp, **DUMP_ARGS)
            if args.verbose:
                print(f"Updated '{file_}'.")


if __name__ == "__main__":
    args = parse_args()
    main(args)
