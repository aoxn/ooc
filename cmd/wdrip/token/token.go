package token

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/cluster-bootstrap/token/util"
	"math"
	"time"
)

type flagpole struct {
	BindAddr     string
	Token        string
	Config       string
	InitialCount int
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Generate new token",
		Long:  "Generate new toke for join node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Token, "new", "", "authentication token")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	token, err := util.GenerateBootstrapToken()
	if err != nil {
		return fmt.Errorf("gen token: %s", err.Error())
	}
	fmt.Println(token)

	return nil
}
func Duration() {
	const (
		base   = 100 * time.Millisecond
		max    = 5 * time.Second
		factor = 2
	)
	duration := base
	for {
		if err := returnError(); err != nil {
			fmt.Printf("start - %d %v\n", duration, err)
			// exponential backoff
			time.Sleep(duration)
			fmt.Printf("after -%d %v\n", duration, err)

			duration = time.Duration(math.Min(float64(max), factor*float64(duration)))
			fmt.Printf("mmmmm - %d %v\n", duration, err)

			continue
		}
		// reset backoff if we have a success
		duration = base

		fmt.Println("finish")
	}
}

func returnError() error {
	return fmt.Errorf("error hahsh")
}
