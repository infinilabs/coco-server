#!/usr/bin/env python3

import os

# All logs start with this identifier
#
# Example: "COCO_TEST_INFO: all tests passed"
LOG_IDENTIFIER = 'COCO_TEST_INFO'

# Run command with `os.system()` then check its exit code.
def sh(command: str):
    status = os.system(command)
    if status != 0:
        print(f"Command [{command}] failed with exit code [{status}], aborting\n", flush=True)
        # print Easysearch logs to make debug easier
        if os.getenv("GITHUB_ACTIONS") == "true":
            print(f"{LOG_IDENTIFIER}: Easysearch logs:\n", flush=True)
            os.system('cat ~/es_install_dir/easysearch.log')
        exit(1)

# Check if we are in the Coco project root by inspecting if the files specified in 
# `files_to_check` exist.
def check_pwd() -> bool:
    files_to_check: list[str] = [
        "README.md",
        "LICENSE",
        "main.go",
        "coco.yml",
        "tests",
        "tests/assets",
        "tests/loadgen.yml"
    ]

    for file in files_to_check:
        if not os.path.exists(file):
            return False

    return True

# Run the tests in the specified DSL file
def run_dsl(dsl_file: str):
    # Cleanup
    sh('bash ./tests/assets/stop_coco.sh')
    sh('bash ./tests/assets/reset_coco_indices.sh')

    sh('bash ./tests/assets/start_coco.sh')
    sh(f'loadgen -config ./tests/loadgen.yml -run {dsl_file} -debug')

    # Cleanup
    sh('bash ./tests/assets/stop_coco.sh')
    sh('bash ./tests/assets/reset_coco_indices.sh')


# Main entry
def run_integration_tests():
    # We should run tests from the root directory, check this
    in_coco_project_root = check_pwd()
    if not in_coco_project_root:
        print(f"{LOG_IDENTIFIER}: {__file__} should be invoked from the project root\n", flush=True)
        exit(1)

    # Discover DSL scenarios under tests/ and execute sequentially
    dsl_files = []
    for root, _, files in os.walk('tests'):
        for file_name in files:
            if file_name.endswith('.dsl'):
                dsl_files.append(os.path.join(root, file_name))

    if not dsl_files:
        print(f'{LOG_IDENTIFIER}: No DSL files found under tests/.\n', flush=True)
        return

    print(f"COCO_TEST_INFO: {len(dsl_files)} tests to run\n", flush=True)
    for dsl_file_index, dsl_file in enumerate(sorted(dsl_files), start=1):
        print(f"{LOG_IDENTIFIER}: Run tests in [{dsl_file_index}:{dsl_file}]\n", flush=True)
        run_dsl(dsl_file)
    print(f"{LOG_IDENTIFIER}: all [{len(dsl_files)}] tests passed!\n", flush=True)


if __name__ == "__main__":
    run_integration_tests()