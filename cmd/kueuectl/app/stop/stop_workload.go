package stop

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/kueue/cmd/kueuectl/app/completion"
	"sigs.k8s.io/kueue/cmd/kueuectl/app/options"
	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
)

var (
	wlLong = templates.LongDesc(`
Puts the given Workload on hold. The Workload will not be admitted and
if it is already admitted it will be put back to queue just as if it 
was preempted (using .spec.active field).
`)
	wlExample = templates.Examples(`
		# Stop the workload 
		kueuectl stop workload my-workload
	`)
)

func NewWorkloadCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	o := options.NewUpdateWorkloadActivationOptions(streams, "stopped", false)

	cmd := &cobra.Command{
		Use: "workload NAME [--namespace NAMESPACE] [--dry-run STRATEGY]",
		// To do not add "[flags]" suffix on the end of usage line
		DisableFlagsInUseLine: true,
		Aliases:               []string{"wl"},
		Short:                 "Stop the Workload",
		Long:                  wlLong,
		Example:               wlExample,
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction:     completion.WorkloadNameFunc(clientGetter, ptr.To(true)),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			err := o.Complete(clientGetter, cmd, args)
			if err != nil {
				return err
			}
			return o.Run(cmd.Context())
		},
	}

	o.PrintFlags.AddFlags(cmd)

	return cmd
}
