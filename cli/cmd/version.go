package cmd

import (
	"fmt"
	"os"

	"github.com/linkerd/linkerd-smi/pkg/version"
	"github.com/spf13/cobra"
)

func newCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stdout, version.Version)
			// TODO: Add Server Version
		},
	}

	return cmd
}
