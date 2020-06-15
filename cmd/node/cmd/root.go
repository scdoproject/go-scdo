/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package cmd

import (
	"fmt"
	"os"

	"github.com/scdoproject/go-scdo/cmd/util"
	"github.com/spf13/cobra"
        "github.com/scdoproject/go-scdo/common"
)

var version bool
// rootCmd represents the base command called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "node",
	Short: "node command for starting a node",
	Long:  `use "node help [<command>]" for detailed usage`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
	Run: func(cmd *cobra.Command, args []string) {
	if version {
	    fmt.Println(common.ScdoNodeVersion)
	  } else {
	    cmd.Help()
	  }
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(util.GetGenerateKeyPairCmd("node"))
	rootCmd.Flags().BoolVarP(&version, "version", "v", false, "print version")

}
