package main

import (
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func main() {
	config := types.Config{
		Quorum: 2,
		Signers: []common.Address{
			common.HexToAddress("0x123"),
			common.HexToAddress("0x456"),
			common.HexToAddress("0x789"),
		},
		GroupSigners: []types.Config{
			{
				Quorum: 5,
				Signers: []common.Address{
					common.HexToAddress("0x123"),
					common.HexToAddress("0x456"),
					common.HexToAddress("0x789"),
					common.HexToAddress("0xabc"),
					common.HexToAddress("0xdef"),
				},
			},
		},
	}
	mcmsContractAddr := "0x123"
	backend := backends.SimulatedBackend{}
	auth := &bind.TransactOpts{}
	configurer := evm.NewConfigurer(backend, auth)
	txHash, err := configurer.SetConfig(mcmsContractAddr, &config, false)
	if err != nil {
		log.Fatalf("failed to set config: %v", err)
	}
	log.Printf("set config tx hash: %s", txHash)
}
