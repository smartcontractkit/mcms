#!/bin/bash

# This script builds the MCM contracts and copies the compiled binaries to the
# destination directory. The destination directory is used by the CTF e2e tests to
# deploy the MCM contracts on Solana.
# Run this script on root level of the repository.
# usage: ./compile-mcm-solana.sh
set -euo pipefail

REPO_URL="https://github.com/smartcontractkit/chainlink-ccip"
REPO_DIR="chainlink-ccip"
MCM_DIR="chains/solana/contracts/programs/mcm"
PROGRAM_DIR="chains/solana/contracts/target/deploy"
DEST_DIR="../artifacts/solana"
TEMP_DIR=$(mktemp -d)
COMMIT_HASH="a91ea5187123329c28553b884e31e5fce4f0e030" # 31 Dec 2024

git clone $REPO_URL $TEMP_DIR/$REPO_DIR
cd $TEMP_DIR/$REPO_DIR/$MCM_DIR
git checkout $COMMIT_HASH

cargo build-sbf

cd -

mkdir -p $DEST_DIR
cp -r $TEMP_DIR/$REPO_DIR/$PROGRAM_DIR/* $DEST_DIR/

rm -rf $TEMP_DIR

echo "MCM contracts compiled successfully"
