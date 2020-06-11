/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package core

import (
	"github.com/seelecredoteam/go-seelecredo/common"
)

func (c *core) handleFinalCommitted() error {
	c.logger.Debug("Received a final committed proposal")
	c.startNewRound(common.Big0)
	return nil
}
