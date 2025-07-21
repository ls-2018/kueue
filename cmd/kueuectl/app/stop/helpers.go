package stop

import (
	"github.com/spf13/cobra"
)

func addKeepAlreadyRunningFlagVar(cmd *cobra.Command, p *bool) {
	cmd.Flags().BoolVar(p, "keep-already-running", false,
		"Indicates whether to keep already running workloads.")
}
