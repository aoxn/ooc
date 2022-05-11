// Package version implements the `version` command
package version

import (
	"fmt"
	"github.com/aoxn/wdrip"

	"github.com/spf13/cobra"
)

// NewCommand returns a new cobra.Command for version
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "version",
		Short: "prints the wdrip CLI version",
		Long:  "prints the wdrip CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(wdrip.Version)
			return nil
		},
	}
	return cmd
}
