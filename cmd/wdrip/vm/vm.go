//go:build darwin
// +build darwin

package vm

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/utils"
	"github.com/aoxn/wdrip/pkg/utils/vz"
	kit "github.com/moby/hyperkit/go"
	"github.com/pkg/errors"
	//"github.com/pkg/term/termios"
	"github.com/google/goterm/term"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type flagVM struct {
	cpu       uint
	mem       uint64
	kernel    string
	initrd    string
	disk      []string
	cmdline   string
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
	cmd.Flags().StringVarP(&flags.kernel, "kernel", "k", "", "kernel file path")
	cmd.Flags().StringVarP(&flags.initrd, "initrd", "i", "", "initrd file path")
	// Stop in the initial ramdisk before attempting
	// to transition to the root file system.
	// Use the first virtio console device as system console.
	cmd.Flags().StringVarP(&flags.cmdline, "arguments", "a", "console=hvc0,root=/dev/vda", "command line arguments for kernel")
	cmd.Flags().StringArrayVarP(&flags.disk, "disk", "d", []string{}, "disk image path")
	cmd.Flags().StringArrayVarP(&flags.publishes, "publish", "p", []string{}, "publish port")

	cmd.AddCommand(NewProxyCommand())
	return cmd
}

func runE(flags *flagVM, cmd *cobra.Command, args []string) error {

	disk := &kit.RawDisk{
		Size: 10 * 1024,
		Path: "/Users/aoxn/vaoxn/code/wdrip/data.img",
	}
	klog.Infof("begin to ensure disk[%s]", disk.Path)
	err := disk.Ensure()
	if err != nil {
		klog.Errorf("error ensure disk[%s]: %s", disk.Path, err.Error())
		return err
	}
	klog.Infof("ensure disk finished")
	return nil
}

//func setRawMode(f *os.File) {
//	var attr unix.Termios
//
//	// Get settings for terminal
//	termios.Tcgetattr(f.Fd(), &attr)
//
//	// Put stdin into raw mode, disabling local echo, input canonicalization,
//	// and CR-NL mapping.
//	attr.Iflag &^= syscall.ICRNL
//	attr.Lflag &^= syscall.ICANON | syscall.ECHO
//
//	// Set minimum characters when reading = 1 char
//	attr.Cc[syscall.VMIN] = 1
//
//	// set timeout when reading as non-canonical mode
//	attr.Cc[syscall.VTIME] = 0
//	termios
//	// reflects the changed settings
//	termios.Tcsetattr(f.Fd(), termios.TCSANOW, &attr)
//}

type Config struct {
	CPU         uint
	Mem         uint64
	Kernel      string
	Initrd      string
	CommandLine string
	Disks       []kit.RawDisk
	Publishes   []*Publish
	PidFile     string
	Pid         int
	State       string
}

func buildConfig(f *flagVM) (*Config, error) {
	cfg := &Config{
		CPU: f.cpu,
		Mem: f.mem,
	}
	if f.kernel == "" {
		return cfg, fmt.Errorf("empty kernel file")
	}
	cfg.Kernel = f.kernel
	if f.initrd == "" {
		return cfg, fmt.Errorf("empty initrd file")
	}
	cfg.Initrd = f.initrd
	if f.cmdline == "" {
		return cfg, fmt.Errorf("empty commandline arguments")
	}
	cfg.CommandLine = strings.Join(strings.Split(f.cmdline, ","), " ")
	for _, d := range f.disk {
		cfg.Disks = append(cfg.Disks, kit.RawDisk{Path: d})
	}
	for _, p := range f.publishes {
		cfg.Publishes = append(cfg.Publishes, NewPublish(p))
	}
	return cfg, nil
}

func create(f *flagVM) error {

	cfg, err := buildConfig(f)
	if err != nil {
		return errors.Wrapf(err, "build vm config from user")
	}
	fmt.Println(utils.PrettyJson(cfg))
	bootLoader := vz.NewLinuxBootLoader(
		cfg.Kernel, vz.WithCommandLine(cfg.CommandLine), vz.WithInitrd(cfg.Initrd),
	)

	klog.Infof("bootLoader: %s", bootLoader.String())

	config := vz.NewVirtualMachineConfiguration(bootLoader, cfg.CPU, cfg.Mem*1024*1024)

	ty, _ := term.Attr(os.Stdin)
	//if err != nil  {
	//	return errors.Wrapf(err, "terminos get")
	//}
	ty.Raw()
	err = ty.Set(os.Stdin)
	//if err != nil {
	//	return errors.Wrapf(err, "terminos set raw")
	//}

	// console
	consoles := []*vz.VirtioConsoleDeviceSerialPortConfiguration{
		vz.NewVirtioConsoleDeviceSerialPortConfiguration(vz.NewFileHandleSerialPortAttachment(os.Stdin, os.Stdout)),
	}
	config.SetSerialPortsVirtualMachineConfiguration(consoles)

	// lnetwork
	//bridge,err := NewBridge()
	//if err != nil {
	//	return errors.Wrapf(err, "new bridge")
	//}
	networkCfg := vz.NewVirtioNetworkDeviceConfiguration(vz.NewNATNetworkDeviceAttachment())
	networkCfg.SetMacAddress(vz.NewRandomLocallyAdministeredMACAddress())
	network := []*vz.VirtioNetworkDeviceConfiguration{networkCfg}
	config.SetNetworkDevicesVirtualMachineConfiguration(network)

	// entropy
	entropyConfig := []*vz.VirtioEntropyDeviceConfiguration{
		vz.NewVirtioEntropyDeviceConfiguration(),
	}
	config.SetEntropyDevicesVirtualMachineConfiguration(entropyConfig)
	var disks []vz.StorageDeviceConfiguration
	for _, d := range cfg.Disks {
		attach, err := vz.NewDiskImageStorageDeviceAttachment(d.Path, false)
		if err != nil {
			return errors.Wrapf(err, "new disk image: %s", d.Path)
		}
		storageDeviceConfig := vz.NewVirtioBlockDeviceConfiguration(attach)
		disks = append(disks, storageDeviceConfig)
	}
	config.SetStorageDevicesVirtualMachineConfiguration(disks)

	// traditional memory balloon device which allows for managing guest memory. (optional)
	ballon := []vz.MemoryBalloonDeviceConfiguration{
		vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration(),
	}
	config.SetMemoryBalloonDevicesVirtualMachineConfiguration(ballon)

	// socket device (optional)
	config.SetSocketDevicesVirtualMachineConfiguration([]vz.SocketDeviceConfiguration{vz.NewVirtioSocketDeviceConfiguration()})
	validated, err := config.Validate()
	if !validated || err != nil {
		panic(fmt.Sprintf("validation failed: %s", err))
	}

	vm := vz.NewVirtualMachine(config)
	//
	if len(vm.SocketDevices()) <= 0 {
		panic("vsock count=0")
	}
	for _, pub := range cfg.Publishes {
		vso := vm.SocketDevices()[0]
		pub.SetStreamHandler(NewStreamVSOCK(vso, Port(pub.raddr)))
		klog.Infof("publish port %d, %s in socket device: %s", pub.laddr, pub.raddr, vso.Ptr())
		proxy := func() {
			recover()
			if err := pub.Listen(); err != nil {
				klog.Errorf("publish %s error: %s", err)
			}
		}
		go proxy()
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM)

	errCh := make(chan error, 1)

	vm.Start(func(err error) {
		if err != nil {
			errCh <- err
		}
	})

	for {
		select {
		case <-signalCh:
			result, err := vm.RequestStop()
			if err != nil {
				return errors.Wrapf(err, "request stop error:")
			}
			klog.Infof("recieved signal", result)
		case newState := <-vm.StateChangedNotify():
			if newState == vz.VirtualMachineStateRunning {
				klog.Infof("start VM is running")
			}
			if newState == vz.VirtualMachineStateStopped {
				return errors.Wrapf(err, "stopped successfully")
			}
		case err := <-errCh:
			klog.Infof("in start:", err)
		}
	}

	// vm.Resume(func(err error) {
	// 	fmt.Println("in resume:", err)
	// })
}
