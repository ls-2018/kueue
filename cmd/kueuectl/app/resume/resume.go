package resume

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"

	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
)

var (
	resumeExample = templates.Examples(`
		# Resume the workload 
		kueuectl resume workload my-workload
	`)
)

func NewResumeCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resume",
		Short:   "Resume the resource",
		Example: resumeExample,
	}

	util.AddDryRunFlag(cmd)

	cmd.AddCommand(NewWorkloadCmd(clientGetter, streams))
	cmd.AddCommand(NewLocalQueueCmd(clientGetter, streams))
	cmd.AddCommand(NewClusterQueueCmd(clientGetter, streams))

	return cmd
}
