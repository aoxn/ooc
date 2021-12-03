// Package version implements the `version` command
package version

import (
	"fmt"
	"github.com/aoxn/ovm"

	"github.com/spf13/cobra"
)

// NewCommand returns a new cobra.Command for version
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "version",
		Short: "prints the ovm CLI version",
		Long:  "prints the ovm CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(ovm.Version)
			return nil
		},
	}
	return cmd
}
