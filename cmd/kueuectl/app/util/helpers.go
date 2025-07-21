package util

import "github.com/spf13/cobra"

func AddAllNamespacesFlagVar(cmd *cobra.Command, p *bool) {
	cmd.Flags().BoolVarP(p, "all-namespaces", "A", false,
		"If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
}
