package mcms

import (
	"github.com/spf13/cobra"
)

func BuildMCMSCmd() *cobra.Command {
	var (
		proposalPath  string
		chainSelector uint64
	)

	cmd := cobra.Command{
		Use:   "mcms",
		Short: "Manage MCMS proposals",
		Long:  ``,
	}

	cmd.PersistentFlags().StringVar(&proposalPath, "proposal", "", "Absolute file path containing the proposal to be submitted")
	cmd.MarkFlagRequired("proposal")
	cmd.PersistentFlags().Uint64Var(&chainSelector, "selector", 0, "Chain selector for the command to connect to")

	cmd.AddCommand(buildDeployCmd(chainSelector))
	cmd.AddCommand(buildMCMSCheckQuorumCmd(proposalPath, chainSelector))
	cmd.AddCommand(buildExecuteChainCmd(proposalPath, chainSelector))
	cmd.AddCommand(buildExecuteOperationCmd(proposalPath, chainSelector))
	cmd.AddCommand(buildSetRootCmd(proposalPath, chainSelector))
	cmd.AddCommand(newSignPrivateKeyCmd(proposalPath))
	cmd.AddCommand(newSignLedgerCmd(proposalPath))

	return &cmd
}
