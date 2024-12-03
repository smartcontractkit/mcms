package main

import (
	"fmt"
	"os"

	"github.com/smartcontractkit/mcms/cmd/mcms"
)

func main() {
	rootCmd := mcms.BuildMCMSCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
