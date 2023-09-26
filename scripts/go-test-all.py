#!/usr/bin/env python3

import os
import subprocess

# Get the environment variables given to us by the Makefile
ARGS = os.environ.get('ARGS', '')
EXTRA_ARGS = os.environ.get('EXTRA_ARGS', '')
TEST_PACKAGES = os.environ.get('TEST_PACKAGES', '')
CURRENT_DIR = os.getcwd()

def find_go_modules(directory='.'):
    """ Find all go.mod files in the current directory and subdirectories. """
    go_mod_files = []
    for root, _, files in os.walk(directory):
        if 'go.mod' in files:
            go_mod_files.append(root)
    return go_mod_files

def run_tests_for_module(module):
    """ Run the unit tests for the given module. """
    path = os.path.join(CURRENT_DIR, module)
    os.chdir(path)
    
    print(f"Running unit tests for {path}")

    test_command = f'go test -mod=readonly {ARGS} {EXTRA_ARGS} {TEST_PACKAGES} ./...'
    result = subprocess.run(test_command, shell=True)
    return result.returncode


def run_tests(directory):
    """ Run the unit tests for all modules in dir. """
    print("Starting unit tests")

    # Find all go.mod files and get their directory names
    go_modules = find_go_modules(directory)

    exit_code = 0
    for gomod in sorted(go_modules):
        exit_code = run_tests_for_module(gomod)
    exit(exit_code)

if __name__ == '__main__':
    run_tests(os.getcwd())
