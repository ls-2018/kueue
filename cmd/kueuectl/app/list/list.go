package list

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
)

var (
	listExample = templates.Examples(`
		# List LocalQueue
  		kueuectl list localqueue
	`)
)

func NewListCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams, clock clock.Clock) *cobra.Command {
	cmd := &cobra.Command{
		Use:        "list",
		Short:      "Display resources",
		Example:    listExample,
		SuggestFor: []string{"ps"},
	}

	cmd.AddCommand(NewLocalQueueCmd(clientGetter, streams, clock))
	cmd.AddCommand(NewClusterQueueCmd(clientGetter, streams, clock))
	cmd.AddCommand(NewWorkloadCmd(clientGetter, streams, clock))
	cmd.AddCommand(NewResourceFlavorCmd(clientGetter, streams, clock))
	cmd.AddCommand(NewPodCmd(clientGetter, streams))

	return cmd
}
