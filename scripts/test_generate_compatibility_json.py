"""Regression tests for release-line compatibility matrix selection."""

import argparse
import importlib.util
import io
import json
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "generate_compatibility_json",
    Path(__file__).with_name("generate-compatibility-json.py"),
)
compatibility = importlib.util.module_from_spec(spec)
spec.loader.exec_module(compatibility)


class CompatibilityVersionTests(unittest.TestCase):
    def test_release_lines_use_minimum_patch_versions(self):
        fields = {
            "from_version": "v8.0.0",
            "TestFeature:from_versions": "v8.7.1,v10.0.1",
        }
        for version, expected in (
            ("v8.7.0", False),
            ("v8.7.1", True),
            ("v8.7.2", True),
            ("v8.8.0", False),
            ("v9.0.0", False),
            ("v10.0.0", False),
            ("v10.0.1", True),
            ("v10.0.2", True),
            ("release-v8.7.x", True),
        ):
            with self.subTest(version=version):
                self.assertEqual(
                    compatibility._test_should_be_run("TestFeature", version, fields),
                    expected,
                )

    def test_explicit_skip_overrides_release_line_selection(self):
        fields = {
            "from_version": "v8.0.0",
            "TestFeature:from_versions": "v8.7.1",
            "TestFeature:skip": "true",
        }
        self.assertFalse(
            compatibility._test_should_be_run("TestFeature", "v8.7.2", fields)
        )

    def test_matrix_contains_only_supported_patches_in_both_directions(self):
        metadata = {
            "test_suite": "TestFeatureSuite",
            "tests": ["TestFeature"],
            "fields": {
                "from_version": "v8.7.0",
                "TestFeature:from_versions": "v8.7.1,v10.0.1",
            },
        }
        args = argparse.Namespace(
            file="unused.go",
            release_version="main",
            image="ghcr.io/cosmos/ibc-go-simd",
            relayer="hermes",
            chain="all",
        )
        versions = ["v8.7.0", "v8.7.1", "v8.7.2", "v10.0.0", "v10.0.1", "v10.0.2"]
        output = io.StringIO()
        with (
            patch.object(compatibility, "parse_args", return_value=args),
            patch.object(compatibility, "_build_file_metadata", return_value=metadata),
            patch.object(compatibility, "_get_ibc_go_releases", return_value=versions),
            redirect_stdout(output),
        ):
            compatibility.main()

        pairs = {
            (entry["chain-a"], entry["chain-b"])
            for entry in json.loads(output.getvalue())["include"]
        }
        expected_versions = {"v8.7.1", "v8.7.2", "v10.0.1", "v10.0.2"}
        self.assertEqual(
            pairs,
            {(version, "main") for version in expected_versions}
            | {("main", version) for version in expected_versions},
        )


if __name__ == "__main__":
    unittest.main()
