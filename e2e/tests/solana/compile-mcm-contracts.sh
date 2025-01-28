#!/bin/bash

# This script builds the MCMS related contracts and copy the compiled binaries to the
# destination directory. The destination directory is used by the CTF e2e tests to
# deploy the programs on Solana.

# Usage: ./e2e/tests/solana/compile-mcm-contracts.sh

set -euo pipefail

REPO_URL="https://github.com/smartcontractkit/chainlink-ccip"
REPO_DIR="chainlink-ccip"
PROJECT_ROOT="$(dirname "$(realpath "$0")")/../../.."
PROGRAM_DIR="chains/solana/contracts/target/deploy"
DEST_DIR="${PROJECT_ROOT}/e2e/artifacts/solana"
TEMP_DIR=$(mktemp -d)
COMMIT_HASH="8a6f4c66925828b35d290510d5a18a5b76a40f33" # 22 Jan 2025

# Programs to build
PROGRAMS=("mcm" "timelock" "access-controller" "external-program-cpi-stub")

cd "${PROJECT_ROOT}"

git clone "${REPO_URL}" "${TEMP_DIR}/${REPO_DIR}"
cd "${TEMP_DIR}/${REPO_DIR}"
git checkout "${COMMIT_HASH}"

for program in "${PROGRAMS[@]}"; do
  cd "chains/solana/contracts/programs/${program}"
  cargo build-sbf
  cd -
done

mkdir -p "${DEST_DIR}"
cp -r "${TEMP_DIR}/${REPO_DIR}/${PROGRAM_DIR}/"* "${DEST_DIR}/"

rm -rf "${TEMP_DIR}"

echo "MCMS contracts compiled successfully"
