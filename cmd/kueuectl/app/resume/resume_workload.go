package resume

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
	wlLong    = templates.LongDesc(`Resumes the Workload, allowing its admission according to regular ClusterQueue rules.`)
	wlExample = templates.Examples(`
		# Resume the workload 
  		kueuectl resume workload my-workload
	`)
)

func NewWorkloadCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	o := options.NewUpdateWorkloadActivationOptions(streams, "resumed", true)

	cmd := &cobra.Command{
		Use: "workload NAME [--namespace NAMESPACE] [--dry-run STRATEGY]",
		// To do not add "[flags]" suffix on the end of usage line
		DisableFlagsInUseLine: true,
		Aliases:               []string{"wl"},
		Short:                 "Resume the Workload",
		Long:                  wlLong,
		Example:               wlExample,
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction:     completion.WorkloadNameFunc(clientGetter, ptr.To(false)),
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
