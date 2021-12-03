package token

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
)

type flagpoleCrypt struct {
	InFile  string
	OutFile string
	IV      string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCryptCommand() *cobra.Command {
	flags := &flagpoleCrypt{}
	cmd := &cobra.Command{
		Use:   "crypt",
		Short: "encrypt/decrypt file",
		Long:  "Generate new toke for join node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCrypt(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.InFile, "in-file", "", "--in-file token.tar")
	cmd.Flags().StringVar(&flags.OutFile, "out-file", "", "--out-file token.tar.data")
	cmd.Flags().StringVar(&flags.IV, "iv", "", "secret iv")
	return cmd
}

func runCrypt(flags *flagpoleCrypt, cmd *cobra.Command, args []string) error {

	if flags.IV == "" {
		flags.IV = MIV
		fmt.Printf("Warning: using default iv, %s\n", MIV)
	}
	if flags.InFile == "" {
		return fmt.Errorf("encrypt file must not be empty")
	}
	data, err := ioutil.ReadFile(flags.InFile)
	if err != nil {
		return fmt.Errorf("read source file: %s", err.Error())
	}
	//
	if flags.OutFile == "" {
		flags.OutFile = fmt.Sprintf("%s.edata", flags.InFile)
	}
	edata, err := Crypt(data, flags)
	if err != nil {
		return fmt.Errorf("encrypt error: %s", err.Error())
	}
	return ioutil.WriteFile(flags.OutFile, edata, 0755)
}

const (
	key = "XfDiMNXOQTPxxNTA"
	MIV = "QQ1875KIEUDLLFMX"
)

func Crypt(data []byte, flag *flagpoleCrypt) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("cipher init error: %s", err.Error())
	}
	cipher.NewCTR(block, []byte(flag.IV)).
		XORKeyStream(data, data)
	return data, nil
}
