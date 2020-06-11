/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package core

import (
	"github.com/seelecredo/go-seelecredo/consensus/istanbul"
)

type backlogEvent struct {
	src istanbul.Validator
	msg *message
}

type timeoutEvent struct{}
