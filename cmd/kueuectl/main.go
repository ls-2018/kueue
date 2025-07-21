package main

import (
	"os"

	"sigs.k8s.io/kueue/cmd/kueuectl/app"
)

func main() {
	if err := app.NewDefaultKueuectlCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
