// Package version implements the `version` command
package version

import (
	"fmt"
	"github.com/aoxn/ooc"

	"github.com/spf13/cobra"
)

// NewCommand returns a new cobra.Command for version
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "version",
		Short: "prints the ooc CLI version",
		Long:  "prints the ooc CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(ooc.Version)
			return nil
		},
	}
	return cmd
}
