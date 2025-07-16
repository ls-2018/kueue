package create

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"

	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
)

var (
	createExample = templates.Examples(`
		# Create local queue 
  		kueuectl create localqueue my-local-queue -c my-cluster-queue
	`)
)

func NewCreateCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a resource",
		Example: createExample,
	}

	util.AddDryRunFlag(cmd)

	cmd.AddCommand(NewLocalQueueCmd(clientGetter, streams))
	cmd.AddCommand(NewClusterQueueCmd(clientGetter, streams))
	cmd.AddCommand(NewResourceFlavorCmd(clientGetter, streams))

	return cmd
}
