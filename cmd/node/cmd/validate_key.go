/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/scdoproject/go-scdo/crypto"
	"github.com/spf13/cobra"
)

var (
	privateKey *string
)

// validatekeyCmd represents the validatekey command
var validatekeyCmd = &cobra.Command{
	Use:   "validatekey",
	Short: "validate the private key and generate its address in shard 1",
	Long: `For example:
			node.exe validatekey`,
	Run: func(cmd *cobra.Command, args []string) {
		key, err := crypto.LoadECDSAFromString(*privateKey)
		addr, err := crypto.GetAddress(&key.PublicKey, uint(1))
		if err != nil {
			fmt.Printf("failed to load the private key: %s\n", err.Error())
			return
		}

		fmt.Printf("Account: %s\n", addr.Hex())
	},
}

func init() {
	rootCmd.AddCommand(validatekeyCmd)

	privateKey = validatekeyCmd.Flags().StringP("key", "k", "", "private key")
	validatekeyCmd.MustMarkFlagRequired("key")
}
