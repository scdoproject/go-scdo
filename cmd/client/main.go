/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package main

import (
	"log"
	"os"

	"github.com/seelecredoteam/go-seelecredo/cmd/client/cmd"
)

func main() {
	app := cmd.NewApp(true)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
