package delete

import (
	"fmt"
	"io"

	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
)

func printWithDryRunStrategy(out io.Writer, name string, dryRunStrategy util.DryRunStrategy) {
	switch dryRunStrategy {
	case util.DryRunClient:
		fmt.Fprintf(out, "%s deleted (client dry run)\n", name)
	case util.DryRunServer:
		fmt.Fprintf(out, "%s deleted (server dry run)\n", name)
	default:
		fmt.Fprintf(out, "%s deleted\n", name)
	}
}
