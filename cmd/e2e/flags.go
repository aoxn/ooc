package e2e

import (
	"flag"
	"testing"
)

type Flags struct {
}

var TestContext = Flags{}

func RegisterCommonFlags() {
	//flag.StringVar(&TestContext.KubeConfig, "kubeconfig", "", "kubernetes config path")
}

// ViperizeFlags sets up all flag and config processing. Future configuration info should be added to viper, not to flags.
func ViperizeFlags() {
	testing.Init()
	// Part 1: Set regular flags.
	// TODO: Future, lets eliminate e2e 'flag' deps entirely in favor of viper only,
	// since go test 'flag's are sort of incompatible w/ flag, glog, etc.
	RegisterCommonFlags()
	flag.Parse()
}
