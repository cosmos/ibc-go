#!/usr/bin/env python3

import json
import os
from typing import Dict


def _load_test_matrix() -> Dict:
    with open("scripts/test-matrix.json", "r") as f:
        return json.loads(f.read())


def main():
    matrix = _load_test_matrix()
    tests = matrix["tests"]
    for test in tests:
        name, binary, image, tags = test["name"], test["binary"], test["image"], test["tags"]
        for tag in tags:
            chain_a, chain_b = tag["chain-a"], tag["chain-b"]
            json_args = json.dumps({
                "test-entry-point": name,
                "chain-a-tag": chain_a,
                "chain-b-tag": chain_b,
                "chain-image": image,
                "chain-binary": binary,
            })

            print(json_args)
            res = os.popen(f'gh workflow run "Manual E2E" --json "{json_args}"').read()
            print(res)


if __name__ == "__main__":
    main()
