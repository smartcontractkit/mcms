package main

import (
	"fmt"
	"os"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

func main() {
	abiPath := os.Args[1]
	binPath := os.Args[2]
	className := os.Args[3]
	pkgName := os.Args[4]
	fmt.Println("Generating", className, "contract wrapper")
	out := className + ".go"

	bindings.Abigen(bindings.AbigenArgs{
		Bin: binPath, ABI: abiPath, Out: out, Type: className, Pkg: pkgName,
	})
}
