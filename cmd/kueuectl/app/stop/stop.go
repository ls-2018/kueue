package stop

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"

	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
)

var (
	stopExample = templates.Examples(`
		# Stop the workload
		kueuectl stop workload my-workload
	`)
)

func NewStopCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop",
		Short:   "Stop the resource",
		Example: stopExample,
	}

	util.AddDryRunFlag(cmd)

	cmd.AddCommand(NewWorkloadCmd(clientGetter, streams))
	cmd.AddCommand(NewLocalQueueCmd(clientGetter, streams))
	cmd.AddCommand(NewClusterQueueCmd(clientGetter, streams))

	return cmd
}
