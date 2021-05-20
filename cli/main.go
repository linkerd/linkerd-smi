package main

import (
	"os"

	"github.com/linkerd/linkerd-smi/cli/cmd"
)

func main() {
	rootCmd := cmd.NewCmdSMI()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
