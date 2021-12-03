package monkey

import (
	"github.com/spf13/cobra"
)

const mhelp = `
chaos monkey jump up.
`

func NewCommand() *cobra.Command {
	//flags := &api.OvmOptions{}
	cmd := &cobra.Command{
		Use:   "monkey",
		Short: "monkey chaos mapper",
		Long:  mhelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE()
		},
	}
	return cmd
}

func runE() error {
	ChaosMonkeyJump()
	return nil
}
