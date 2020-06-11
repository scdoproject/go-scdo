/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package core

import (
	"github.com/seelecredoteam/go-seelecredo/consensus/istanbul"
)

type backlogEvent struct {
	src istanbul.Validator
	msg *message
}

type timeoutEvent struct{}
