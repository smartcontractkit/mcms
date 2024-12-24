#!/usr/bin/env sh

set -e

admin=$(solana address)
ledger_dir="/tmp/solana-test-validator-ledger"

rm -rf "${ledger_dir}"
mkdir -p "${ledger_dir}"

exec solana-test-validator \
  --reset \
  --rpc-port 8999 \
  --ledger "${ledger_dir}" \
  --upgradeable-program 6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/mcm.so                       ${admin} \
  --upgradeable-program LoCoNsJFuhTkSQjfdDfn3yuwqhSYoPujmviRHVCzsqn  ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/timelock.so                  ${admin} \
  --upgradeable-program 9xi644bRR8birboDGdTiwBq3C7VEeR7VuamRYYXCubUW ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/access_controller.so         ${admin} \
  --upgradeable-program CtEVnHsQzhTNWav8skikiV2oF6Xx7r7uGGa8eCDQtTjH ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/ccip_receiver.so             ${admin} \
  --upgradeable-program 9Vjda3WU2gsJgE4VdU6QuDw8rfHLyigfFyWs3XDPNUn8 ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/ccip_invalid_receiver.so     ${admin} \
  --upgradeable-program C8WSPj3yyus1YN3yNB6YA5zStYtbjQWtpmKadmvyUXq8 ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/ccip_router.so               ${admin} \
  --upgradeable-program GRvFSLwR7szpjgNEZbGe4HtxfJYXqySXuuRUAJDpu4WH ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/token_pool.so                ${admin} \
  --upgradeable-program 4HeqEoSyfYpeC2goFLj9eHgkxV33mR5G7JYAbRsN14uQ ${HOME}/cll/chainlink-ccip/chains/solana/contracts/target/deploy/external_program_cpi_stub.so ${admin}
