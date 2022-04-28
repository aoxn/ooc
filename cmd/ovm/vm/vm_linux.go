//go:build linux
// +build linux

package vm

import (
	"fmt"
	"github.com/spf13/cobra"
)

type flagVM struct {
	cpu     uint
	mem     uint64
	kernel  string
	initrd  string
	disk    []string
	cmdline string
	publishes []string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := flagVM{}
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "vm related",
		Long:  "unknown",
		RunE: func(cmd *cobra.Command, args []string) error {

			return create(&flags)
		},
	}
	cmd.Flags().UintVarP(&flags.cpu, "cpu", "c", 1, "cpu amount, core")
	cmd.Flags().Uint64VarP(&flags.mem, "mem", "m", 1024, "memory amount, MB")
	cmd.Flags().StringVarP(&flags.kernel, "kernel", "k","", "kernel file path")
	cmd.Flags().StringVarP(&flags.initrd, "initrd", "i", "","initrd file path")
	// Stop in the initial ramdisk before attempting
	// to transition to the root file system.
	// Use the first virtio console device as system console.
	cmd.Flags().StringVarP(&flags.cmdline, "arguments", "a", "console=hvc0,root=/dev/vda", "command line arguments for kernel")
	cmd.Flags().StringArrayVarP(&flags.disk, "disk", "d", []string{}, "disk image path")
	cmd.Flags().StringArrayVarP(&flags.publishes, "publish", "p", []string{}, "publish port")

	cmd.AddCommand(NewProxyCommand())
	return cmd
}
func create(vm *flagVM) error {
	return fmt.Errorf("unimplemented")
}
